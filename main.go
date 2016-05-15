package main

import (
	"fmt"
	"log"
	"sync"
	"ruscraper/conf"
	"time"
	"net/http"
	"strings"
	"strconv"
	"io/ioutil"
	"unicode/utf8"
	"encoding/json"
	"gopkg.in/redis.v3"
	"gopkg.in/olivere/elastic.v3"
	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/charmap"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
)

type Theme struct {
	Id int64
	Read bool
	Name string
	Size string
	Date string
	Answers string
}

type Page struct {
	Number int
	Themes []Theme
}

type FuncUnit struct {
	Redis *redis.Client
	Elastic *elastic.Client
} 

var funcUnit = FuncUnit{}

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

func ParsePage(baseUrl string, p int, pageChan chan Page) {

	fmt.Println("parse page -> ", baseUrl, p)	
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

	columnCnt := 0

	var theme Theme

	if doc == nil {
		fmt.Printf("page %d is nil\r\n", p)
		pageChan <- Page{p, []Theme{}}
		return
	}

	fmt.Println(doc)

	doc.Find(".forumline tr.hl-tr td").Each(func(i int, s *goquery.Selection) {

		title := s.Text()
		decodedTitle := strings.Replace(fromCharmap(title), "\r\n", "", -1)
		if columnCnt % 5 == 0 {
			theme = Theme{}
			id_str, _ := s.Attr("id")
			id, _ := strconv.Atoi(id_str)
			theme.Id = int64(id)
			columnCnt = 0
		} else {
			if decodedTitle != "" {
				if columnCnt == 1 {
					theme.Name = decodedTitle
				} else if columnCnt == 2 {
					theme.Size = strings.Replace(fromCharmap(title), "\t", "", -1)
				} else if columnCnt == 3 {
					theme.Date = strings.Replace(fromCharmap(title), "\t", "", -1)
				} else if columnCnt == 4 {
					theme.Answers = strings.Replace(fromCharmap(title), "\t", "", -1)
					themes = append(themes, theme)
				}
			}
		}

		columnCnt += 1

	})

	fmt.Println("find rows ->", len(themes))
	//fmt.Println(themes)	
	page := Page{p, themes}

	//add themes to elastic search
		
	for _, t := range(page.Themes) {

		termQuery := elastic.NewTermQuery("name", "test")

		searchResult, err := funcUnit.Elastic.Search().
		    Index("programming_videos").
		    Query(termQuery).   // specify the query
		    Sort("name", true). // sort by "user" field, ascending
		    From(0).Size(1).   // take documents 0-9
		    Pretty(true).       // pretty print request and response JSON
		    Do()                // execute

		if err != nil {
		    // Handle error
		    fmt.Println("elastic - search fail", err)
		    // panic(err)
		}

		t.Read = true

		if searchResult == nil || searchResult.TotalHits() != int64(0) {
			funcUnit.Redis.Incr(time.Now().Format("00060101") + "_new_themes_count")
			t_id := strconv.Itoa(int(t.Id))
			_, err = funcUnit.Elastic.Index().
			    Index("programming_videos").
			    Type("theme").
			    Id(t_id).
			    BodyJson(t).
			    Do()

			if err != nil {
			    // Handle error
			    panic(err)
			}

			t.Read = false
		} else {
			fmt.Println("skip")
		}
	}

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

func RunParse(url string, pagesCnt int, parsedChan chan []Page) {

	pageChan := make(chan Page)	
	var wg sync.WaitGroup

	wg.Add(pagesCnt)

	go ReadData(pagesCnt, pageChan, &wg, parsedChan)
	for p := 0; p < pagesCnt; p++ {
		go ParsePage(url, p, pageChan)
	}

	wg.Wait()
	fmt.Println("return result")
}

type ParseUrl struct {
	Url string `json:"url"`
}

func main() {

	router := gin.Default()

	// router.GET("/themes", func(c *gin.Context) {
	// 	c.JSON(200, gin.H{
	// 		search := c.QueryString('search')
	// 		term := c.QueryString('term')

	// 	})
	// })

	config := conf.Conf{}
	config.Read()

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

    funcUnit.Redis = redisC

    //ELASTIC

    client, err := elastic.NewClient()
	if err != nil {
    	// Handle error
	}

	fmt.Println("elastic - ok")
	funcUnit.Elastic = client

	_, err = client.CreateIndex("programming_videos").Do()
	if err != nil {
    	// Handle error  
		if !strings.Contains(fmt.Sprintf("%s", err), "index_already_exists_exception") {
			log.Fatalf("elastic - CreateIndex", err)
    		panic(err)
		}
	}
	_, err = client.CreateIndex("programming_books").Do()
	if err != nil {
    	// Handle error
    	if !strings.Contains(fmt.Sprintf("%s", err), "index_already_exists_exception") {
    		log.Fatalf("elastic - CreateIndex", err)
    		panic(err)
		}
	}

	router.GET("/stat", func(c *gin.Context) {

		parseAttemps, _ := redisC.Get("parse_attemps").Result()

		c.JSON(200, gin.H{
			"parse_attemps": parseAttemps,
			"redisStat": "{}",
		})
	})

	router.GET("/parse_urls", func(c *gin.Context) {
		fmt.Println("GET urls", config.ParseUrls)
		c.JSON(200, gin.H{
			"parse_urls": config.ParseUrls,
		})
	})

	router.POST("/parse", func(c *gin.Context) {

		pages := c.DefaultPostForm("pages", fmt.Sprintf("%d",10))

		var params ParseUrl
		c.Bind(&params)
		urlParse := params.Url
		fmt.Printf("start parse=%s\r\n", urlParse)
		err := redisC.Incr("parse_attemps").Err()

		if err != nil {
			fmt.Println("error on redis")
		}

		//pagesJson := []interface{}{}
		//themesJson := []interface{}{}

		pagesNum, _ := strconv.Atoi(pages)
		parsedChan := make (chan []Page)
		//parse pages
		RunParse(urlParse, pagesNum, parsedChan)

		pagesParsed := <- parsedChan

		fmt.Printf("parse ok %s\r\n", pagesParsed)

		list := gin.H{}

		for k, p := range(pagesParsed) {
			pgs, _ := json.Marshal(p)
			fmt.Printf("encode ok %s\r\n", pgs)
			// pagesJson = append(pagesJson, pgs)
			for c, t := range(p.Themes) {
				//tjs, _ := json.Marshal(t)
				// themesJson = append(themesJson, tjs)
				list[strconv.Itoa(c * k)] = t
			}
		}

		c.JSON(200, list)
	})

	router.Run()
}