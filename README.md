# novelSpider
一个可以自定义请求等待时间与Headers的、带有本地网页缓存的、轻便的小说爬虫框架

## 食用方法

1. 在项目目录下使用`go get github.com/PuerkitoBio/goquery`获取该框架并在项目中导入该框架；
1. 使用函数`novelSpider.NewNovelSpider`创建一个新爬虫；
1. 分别设置这个新爬虫的三个闭包`GetChapterName`、`GetChapterContent`、`GetNextChapterLink`，在闭包中用jQuery语法调用GoQuery框架或正则表达式等你喜欢的任何方式来返回章节名、小说正文和下一章的链接；
1. 等待爬取结束。

## 样板代码

```
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/DOUBLEU9264/novelSpider"
	"github.com/PuerkitoBio/goquery"
)

func main() {
	args := os.Args
	if len(args) < 4 {
		fmt.Println("使用书名、起始章节和链接作为参数重新运行")
		return
	}

	bookName := args[1] // 书名
	chapterIndex, err := strconv.Atoi(args[2])
	if err != nil || chapterIndex < 0 {
		log.Println("起始章节不能为复数")
		return
	}
	url := args[3]

	userAgent := `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36`
	destPath := "./dest/"        // 目标路径
	cachePath := "./html-cache/" // 缓存路径
	re := regexp.MustCompile(`[  ]+`)

	ns := novelSpider.NewNovelSpider(bookName, url, destPath, cachePath, chapterIndex, 5)

	ns.Headers = map[string]string{"User-Agent": userAgent}

	ns.GetChapterName = func(dom *goquery.Document) (string, error) {
		chapterName := dom.Find("div.nr_function > h1").First().Text()
		return strings.TrimSpace(chapterName), nil
	}

	ns.GetChapterContent = func(dom *goquery.Document) (string, error) {
		content := strings.TrimSpace(dom.Find("div.novelcontent").First().Text())
		return re.ReplaceAllString(content, ""), nil
	}

	ns.GetNextChapterLink = func(dom *goquery.Document) (string, error) {
		nextUrl, ok := dom.Find("div.page_chapter > ul > li > a.p4").Attr("href")
		if !ok {
			return "", fmt.Errorf("无法获取下一章链接: %s", url)
		}
		return nextUrl, nil
	}

	if err := ns.Run(); err != nil {
		log.Println(err)
	}
}
```