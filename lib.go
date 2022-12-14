package novelSpider

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type NovelSpider struct {
	BookName            string                                      // 小说名
	URL                 string                                      // 爬虫起始网址
	DestPath            string                                      // 目标文件夹
	CachePath           string                                      // 网页缓存文件夹
	CurrentChapterIndex int                                         // 当前章节数
	GetChapterName      func(dom *goquery.Document) (string, error) // 获取页面中的章节名
	GetChapterContent   func(dom *goquery.Document) (string, error) // 获取页面中本章正文内容
	GetNextChapterLink  func(dom *goquery.Document) (string, error) // 获取页面中下一章的链接
	Headers             map[string]string                           // 自定义请求头
	destFile            *os.File                                    // 待写入的目标文件
	siteDomain          string                                      // 网站域名
	requestWaitTime     time.Duration
}

func NewNovelSpider(bookName, url, destPath, cachePath string,
	currentChapterIndex, requestWaitSec int,
) NovelSpider {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	siteDomain, _, _ := strings.Cut(url, "/")

	return NovelSpider{
		BookName:            bookName,
		URL:                 url,
		DestPath:            destPath,
		CachePath:           path.Join(cachePath, bookName),
		CurrentChapterIndex: currentChapterIndex,
		siteDomain:          siteDomain,
		requestWaitTime:     time.Duration(requestWaitSec) * time.Second,
	}
}

func (ns *NovelSpider) Run() error {
	// 生成所需文件夹
	os.MkdirAll(ns.CachePath, 0755)
	os.MkdirAll(ns.DestPath, 0755)

	// 打开目标文件
	var err error
	ns.destFile, err = os.OpenFile(path.Join(ns.DestPath, ns.BookName+".txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln("打开目标文件失败:", err)
	}
	defer ns.destFile.Close()

	// 爬取文章
	log.Println(ns.BookName, ns.URL, "爬虫开始")
	err = ns.spide(ns.URL)
	if err != nil {
		return fmt.Errorf("%v 爬虫终止", err)
	}
	return nil
}

func (ns *NovelSpider) spide(url string) error {
	dom, err := ns.cacheOrRequest(url, path.Base(url))
	if err != nil {
		return err
	}

	chapterName, err := ns.GetChapterName(dom)
	if err != nil {
		log.Printf("第 %d 章 URL: %s, 获取章节名失败: %v", ns.CurrentChapterIndex, url, err)
	}

	content, err := ns.GetChapterContent(dom)
	if err != nil {
		return fmt.Errorf("第 %d 章 URL: %s, 无法获取正文内容: %v", ns.CurrentChapterIndex, url, err)
	}

	ns.destFile.WriteString(fmt.Sprintf("第 %d 章 %s\n\n%s\n\n", ns.CurrentChapterIndex, chapterName, content))

	log.Printf("第 %d 章 %s 完成\n", ns.CurrentChapterIndex, chapterName)

	nextUrl, err := ns.GetNextChapterLink(dom)
	if err != nil {
		return fmt.Errorf("第 %d 章 URL: %s, 无法获取下一章链接: %v", ns.CurrentChapterIndex, url, err)
	}
	nextUrl = path.Join(ns.siteDomain, nextUrl)

	ns.CurrentChapterIndex++

	time.Sleep(ns.requestWaitTime)
	return ns.spide(nextUrl)
}

// 读取缓存文件，不存在则请求网址并建立缓存
func (ns *NovelSpider) cacheOrRequest(url, fileName string) (*goquery.Document, error) {
	filePath := path.Join(ns.CachePath, fileName)

	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// 查询缓存
	f := ns.getCache(filePath)
	if f == nil {
		log.Printf("发送请求: %s\n", url)

		// 缓存不存在，发送请求
		resp, err := ns.requestWithCostumHeaders(url, ns.Headers)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// 检查状态码
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("状态码错误")
		}

		// 创建缓存
		f, err = ns.makeCache(resp.Body, filePath)
		if err != nil {
			return nil, err
		}
	} else {
		log.Printf("使用%s的缓存: %s\n", url, filePath)
	}

	defer f.Close()
	return goquery.NewDocumentFromReader(f)
}

// 获取缓存文件，存在则打开文件，否则返回nil
func (ns *NovelSpider) getCache(filePath string) *os.File {

	f, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		log.Printf("打开缓存文件失败：%v\n", err)
		return nil
	}

	return f
}

func (ns *NovelSpider) requestWithCostumHeaders(url string, headers map[string]string) (*http.Response, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return client.Do(req)
}

// 为一个readcloser创建文件缓存，返回缓存文件
func (ns *NovelSpider) makeCache(rc io.ReadCloser, filePath string) (*os.File, error) {

	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	_, err = f.Write(b)
	if err != nil {
		return nil, err
	}

	f.Seek(0, 0)
	return f, nil
}
