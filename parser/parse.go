package parser

import (
	"fmt"
	"sync"
	"strings"
	"strconv"
	"regexp"
	"io/ioutil"
	"unicode/utf8"
	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/charmap"
	"github.com/PuerkitoBio/goquery"
	"ruscraper/core"
	"ruscraper/models"
	"ruscraper/storage"
	"ruscraper/helpers"
)

func DecodeUtf(str string) []rune {

	runes := []rune{}

	for len(str) > 0 {
		r, size := utf8.DecodeRuneInString(str)
		runes = append(runes, r)
		str = str[size:]
	}

	return runes
}

func FromCharmap(str string) string {

	sr := strings.NewReader(str)
	tr := transform.NewReader(sr, charmap.Windows1251.NewDecoder())
	buf, err := ioutil.ReadAll(tr)
	if err != err {
	 // обработка ошибки
	}

	return string(buf)
}

func ReadData(pagesCnt int, pageChan chan models.Page, wg *sync.WaitGroup, parsedChan chan []models.Page) {

	parsedPages := []models.Page{}

	// timeStart := time.Now().Unix()
	// go func() {
	// 	time.Sleep(1)

	// 	timeEnd	:= time.Now().Unix()
	// 	if timeEnd - timeStart > 60 {
	// 		close(pageChan)
	// 	}
	// }()

	for page := range(pageChan) {
		//fmt.Printf("page %d parsed\r\n", page.Number)
		wg.Done()
		parsedPages = append(parsedPages, page)

		if len(parsedPages) == pagesCnt {
			close(pageChan)
		}
	}

	parsedChan <- parsedPages
}

func RunParse(url string, pagesCnt int, parsedChan chan []models.Page) {

	pageChan := make(chan models.Page)	
	var wg sync.WaitGroup

	wg.Add(pagesCnt)

	go ReadData(pagesCnt, pageChan, &wg, parsedChan)
	for p := 0; p < pagesCnt; p++ {
		go ParsePage(url, p, pageChan)
	}

	wg.Wait()
}

var YearRegexp = regexp.MustCompile(`\[(\d{4})\,.*\]`)

func ParsePage(baseUrl string, p int, pageChan chan models.Page) {

	//fmt.Println("parse page -> ", baseUrl, p)	
	var doc *goquery.Document

	if p > 0 {
		url := baseUrl + "&start=" + strconv.Itoa(p * 50)
		doc, _ = goquery.NewDocument(url)
	} else {
		doc, _ = goquery.NewDocument(baseUrl)
	}

	themes := []models.Theme{}

	columnCnt := 0

	var theme models.Theme

	if doc == nil {
		fmt.Printf("page %d is nil\r\n", p)
		helpers.SetTimeCounter("parse_fail")			
		pageChan <- models.Page{p, []models.Theme{}}
		return
	}

	doc.Find(".forumline tr.hl-tr td").Each(func(i int, s *goquery.Selection) {

		title := s.Text()
		decodedTitle := strings.Replace(FromCharmap(title), "\r\n", "", -1)
		if columnCnt % 5 == 0 {
			theme = models.Theme{}
			id_str, _ := s.Attr("id")
			id, _ := strconv.Atoi(id_str)
			theme.Id = int64(id)
			columnCnt = 0
		} else {
			if decodedTitle != "" {
				if columnCnt == 1 {
					theme.Name = decodedTitle
					publicateYear := YearRegexp.FindString(decodedTitle)
					publicateYear = strings.TrimRight(publicateYear, "]")
					publicateYear = strings.TrimLeft(publicateYear, "[")
					publicateYear = strings.Split(publicateYear, ",")[0]
					if publicateYear != "" {					
						year, _ := strconv.Atoi(publicateYear)
						theme.PubYear = year
					}
				} else if columnCnt == 2 {
					theme.Size = strings.Replace(FromCharmap(title), "\t", "", -1)
				} else if columnCnt == 3 {
					theme.Date = strings.Replace(FromCharmap(title), "\t", "", -1)
				} else if columnCnt == 4 {
					theme.Answers = strings.Replace(FromCharmap(title), "\t", "", -1)
					themes = append(themes, theme)
				}
			}
		}

		columnCnt += 1

	})

	page := models.Page{p, themes}
	storage.SaveToStore(themes, core.Config.UrlToElastic[baseUrl])
	pageChan <- page
}


func StartParse(urlParse string, pages int) (themes []models.Theme) {

	//fmt.Printf("start parse=%s\r\n", urlParse)
	err := core.Units.Redis.Incr("parse_attemps").Err()

	if err != nil {
		fmt.Println("error on redis")
	}

	parsedChan := make (chan []models.Page)
	//parse pages
	RunParse(urlParse, pages, parsedChan)

	pagesParsed := <- parsedChan

	fmt.Printf("parse ok %s\r\n", len(pagesParsed))

	themes = []models.Theme{}

	for _, p := range(pagesParsed) {		
		for _, t := range(p.Themes) {
			themes = append(themes, t)			
		}
	}

	//разблокируем
	err = core.Units.Redis.Set("ruscraper_parse_lock", "", 0).Err()

	if err != nil {
		fmt.Println("error on redis", err)
	}

	return
}