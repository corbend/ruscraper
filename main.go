package main

import (
	"fmt"
	"sync"
	//"time"
	//"log"
	"net/http"
	"strings"
	"strconv"
	"io/ioutil"
	"unicode/utf8"
	"encoding/json"
	"gopkg.in/redis.v3"
	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/charmap"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
)

type Theme struct {
	Name string
	Size string
	Date string
	Answers string
}

type Page struct {
	Number int
	Themes []Theme
}

func DecodeUtf(str string) []rune {

	runes := []rune{}

	for len(str) > 0 {
		r, size := utf8.DecodeRuneInString(str)
		runes = append(runes, r)
		str = str[size:]
	}

	return runes
}

func fromCharmap(str string) string {

	sr := strings.NewReader(str)
	tr := transform.NewReader(sr, charmap.Windows1251.NewDecoder())
	buf, err := ioutil.ReadAll(tr)
	if err != err {
	 // обработка ошибки
	}

	return string(buf)
}

func ParsePage(p int, pageChan chan Page) {

	fmt.Println("parse page -> ", p)
	baseUrl := "http://rutracker.org.unblock.ga/forum/viewforum.php?f=1565"
	var doc *goquery.Document

	if p > 0 {
		url := baseUrl + "&start=" + strconv.Itoa(p * 50)
		fmt.Println(url)
		doc, _ = goquery.NewDocument(url)
	} else {
		doc, _ = goquery.NewDocument(baseUrl)
	}

	// if err != nil {
	// 	log.Fatal(err)
	// }

	themes := []Theme{}

	titleUnicode := "Темы"

	columnCnt := 0
	themeCnt := 1
	themeStartFound := false

	var theme Theme

	if doc == nil {
		fmt.Printf("page %d is nil\r\n", p)
		pageChan <- Page{p, []Theme{}}
		return
	}

	fmt.Println(doc)

	doc.Find(".forumline tr td").Each(func(i int, s *goquery.Selection) {

		title := s.Text()
		decodedTitle := fromCharmap(title)

		if decodedTitle == titleUnicode {
			themeStartFound = true
		}

		if themeCnt > 2 {
			themeStartFound = false

			if columnCnt % 5 == 0 {
				themes = append(themes, theme)				
				theme = Theme{}
				theme.Name = decodedTitle
				columnCnt = 0
			} else {
				if columnCnt == 2 {
					theme.Size = decodedTitle
				} else if columnCnt == 3 {
					theme.Date = decodedTitle
				} else if columnCnt == 4 {
					theme.Answers = decodedTitle
				}
			}

			columnCnt += 1
		}

		if themeStartFound {
			themeCnt += 1
		}

	})

	fmt.Println("find rows ->", len(themes))
	//fmt.Println(themes)	
	page := Page{p, themes}
	pageChan <- page
}

func ReadData(pagesCnt int, pageChan chan Page, wg *sync.WaitGroup, parsedChan chan []Page) {

	parsedPages := []Page{}
	fmt.Println("READ BEGIN")

	// timeStart := time.Now().Unix()
	// go func() {
	// 	time.Sleep(1)

	// 	timeEnd	:= time.Now().Unix()
	// 	if timeEnd - timeStart > 60 {
	// 		close(pageChan)
	// 	}
	// }()

	for page := range(pageChan) {
		fmt.Printf("page %d parsed\r\n", page.Number)
		wg.Done()
		parsedPages = append(parsedPages, page)

		if len(parsedPages) == pagesCnt {
			close(pageChan)
		}
	}

	fmt.Println("READ END")	
	parsedChan <- parsedPages
}

func RunParse(pagesCnt int, parsedChan chan []Page) {

	pageChan := make(chan Page)	
	var wg sync.WaitGroup

	wg.Add(pagesCnt)

	go ReadData(pagesCnt, pageChan, &wg, parsedChan)
	for p := 0; p < pagesCnt; p++ {
		go ParsePage(p, pageChan)
	}

	wg.Wait()
	fmt.Println("return result")
}

func main() {

	router := gin.Default()

	// router.GET("/themes", func(c *gin.Context) {
	// 	c.JSON(200, gin.H{
	// 		search := c.QueryString('search')
	// 		term := c.QueryString('term')

	// 	})
	// })

	router.Static("/assets", "./assets")
	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	//REDIS

	redisC := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })

	router.GET("/stat", func(c *gin.Context) {

		parseAttemps, _ := redisC.Get("parse_attemps").Result()

		c.JSON(200, gin.H{
			"parse_attemps": parseAttemps,
			"redisStat": "{}",
		})
	})

	router.POST("/parse", func(c *gin.Context) {
		fmt.Println("start parse")
		pages := c.DefaultPostForm("pages", fmt.Sprintf("%d",10))

		err := redisC.Incr("parse_attemps").Err()

		if err != nil {
			fmt.Println("error on redis")
		}

		pagesJson := []interface{}{}
		themesJson := []interface{}{}

		pagesNum, _ := strconv.Atoi(pages)
		parsedChan := make (chan []Page)
		RunParse(pagesNum, parsedChan)
		pagesParsed := <- parsedChan

		fmt.Printf("parse ok %s\r\n", pagesParsed)
		for _, p := range(pagesParsed) {
			pgs, _ := json.Marshal(p)
			fmt.Printf("encode ok %s\r\n", pgs)
			pagesJson = append(pagesJson, pgs)
			for _, t := range(p.Themes) {
				tjs, _ := json.Marshal(t)
				themesJson = append(themesJson, tjs)
			}
		}
		
		c.JSON(200, gin.H{
			"success": "true",
			"pages": pagesJson,
			"themes": themesJson,
		})
	})

	router.Run()
}