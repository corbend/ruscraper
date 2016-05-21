package main

import (	
	"fmt"
	"log"
	"time"
	"strings"
	"net/http"
	"strconv"
	"encoding/json"
	"ruscraper/core"
	"ruscraper/queue"
	"ruscraper/scheduler"
	"ruscraper/parser"
	"ruscraper/models"
	"gopkg.in/olivere/elastic.v3"
	"github.com/gin-gonic/gin"
)

func main() {

	core.InitConfig()

	router := gin.Default()

	router.Static("/assets", "./assets")
	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	fmt.Println("Redis Queue - ok")

	router.GET("/stat", func(c *gin.Context) {

		date := time.Now()
		year, month, day := date.Date()
		dateStr := fmt.Sprintf("%d-%d-%dT%d:00:00", year, month, day, date.Hour())

		parseAttemps, _ := core.Units.Redis.Get("parse_attemps").Result()
		runningTasksCnt, _ := core.Units.Redis.Get("running_tasks_cnt").Result()
		newHitsCnt1, _ := core.Units.Redis.Get("new_hits_cnt_" + dateStr + "_programming_books").Result()
		newHitsCnt2, _ := core.Units.Redis.Get("new_hits_cnt_" + dateStr + "_programming_videos").Result()

		lastUpdateTime, _ := core.Units.Redis.Get("new_hits_update_time").Result()

		c.JSON(200, gin.H{
			"parse_attemps": parseAttemps,
			"new_hits_cnt_" + dateStr + "_programming_books": newHitsCnt1,
			"new_hits_cnt_" + dateStr + "_programming_videos": newHitsCnt2,
			"new_hits_update_time": lastUpdateTime,
			"running_tasks_cnt": runningTasksCnt,
			"redisStat": "{}",
		})
	})

	router.GET("/parse_urls", func(c *gin.Context) {
		fmt.Println("GET urls", core.Config.ParseUrls)
		c.JSON(200, gin.H{
			"parse_urls": core.Config.ParseUrls,
		})
	})

	router.POST("/filters", func(c *gin.Context) {
		//save filter to sqlite
		fmt.Println("save filter")
		var newFilter models.ThemeFilter
		c.Bind(&newFilter)

		term_values := []string{}

		for _, term_value := range(strings.Split(c.PostForm("term_values"), ",")) {
			term_values = append(term_values, c.PostForm(term_value))
		}

		if len(term_values) > 0 {
			newFilter.TermValuesList = term_values
		}

		newFilter.SaveToDb()

		c.JSON(200, gin.H{
			"success": true,
		})
	})

	router.POST("/filters/apply", func(c *gin.Context) {
		fmt.Println("save filter")
		var applyFilter models.ThemeFilter
		c.Bind(&applyFilter)

		termQuery := elastic.NewTermQuery(applyFilter.TermName, applyFilter.TermValues)

		searchResult, err := core.Units.Elastic.Search().
		    Index(applyFilter.IndexName).
		    Query(termQuery).   // specify the query
		    Sort(applyFilter.TermName, true). // sort by "user" field, ascending
		    From(0).Size(10).   // take documents 0-9
		    Pretty(true).       // pretty print request and response JSON
		    Do()                // execute

		if err != nil {
		    // Handle error
		    fmt.Println("elastic - search by filter fail", err, applyFilter)
		    // panic(err)
		}

		results := gin.H{}

		if searchResult.Hits != nil {
			for index, hit := range searchResult.Hits.Hits {
		        // hit.Index contains the name of the index

		        // Deserialize hit.Source into a Tweet (could also be just a map[string]interface{}).
		        var t models.Theme
		        err := json.Unmarshal(*hit.Source, &t)
		        if err != nil {
		            // Deserialization failed
		            log.Fatalf("Deserialization failed")
		        }

		        results[strconv.Itoa(index)] = t
		    }
		}

		c.JSON(200, results)
	})

	router.GET("/filters/:filter_name", func(c *gin.Context) {

		filterName, _ = c.Params.Get('filter_name')

		if filterName == 'LastDay' {
			records, _ := models.GetLastThemes(nowDayTime.Unix(), time.Now().Unix()))
		}
		
		results := gin.H{}
		    
	 	for index, r := records {
			results[strconv.Itoa(index)] = r
	 	}

		c.JSON(200, results)
	})

	router.GET("/filters", func(c *gin.Context) {
		c.JSON(200, models.GetAllFilters(c))
	})

	router.POST("/parse", func(c *gin.Context) {

		pages := c.DefaultPostForm("pages", fmt.Sprintf("%d", 10))

		params := models.ParseUrl{}
		c.Bind(&params)
		urlParse := params.Url

		pagesNum, _ := strconv.Atoi(pages)
		list := parser.StartParse(urlParse, pagesNum)

		response := gin.H{}
		for c, t := range(list) {
			response[strconv.Itoa(c)] = t
		}

		c.JSON(200, response)
	})

	router.DELETE("/filters/:id", func(c *gin.Context) {
		filterIdStr, _ := c.Params.Get("id")
		filterId, _ := strconv.Atoi(filterIdStr)
		c.JSON(200, models.RemoveFilter(filterId, c))
	})

	//run parsing in goroutine
	queue.SetupConsumers()
	core.Units.Redis.Set("ruscraper_parse_lock", "", 0).Err()

	date := time.Now()
	year, month, day := date.Date()
	dateStr := fmt.Sprintf("%d-%d-%dT%d:00:00", year, month, day, date.Hour())

	currentHitsCnt1, _ := core.Units.Redis.Get("new_hits_cnt_"+dateStr+"_programming_videos").Result()
	currentHitsCnt2, _ := core.Units.Redis.Get("new_hits_cnt_"+dateStr+"_programming_books").Result()

	if currentHitsCnt1 == "" {
		core.Units.Redis.Set("new_hits_cnt_"+dateStr+"_programming_videos", 0, 0).Err()
	}

	if currentHitsCnt2 == "" {
		core.Units.Redis.Set("new_hits_cnt_"+dateStr+"_programming_books", 0, 0).Err()
	}

	for _, urlToParse := range(core.Config.ParseUrls) {
		scheduler.RunTimer(core.Config.ParseInterval, 		
		func() {
			r, _ := core.Units.Redis.Get("ruscraper_parse_lock").Result()
			if r == "" {
				task := queue.ParseTask{}
				task.Url = urlToParse
				task.Action = "start"
				task.NumPages = core.Config.ParsePagesNum
				task.IndexName = core.Config.UrlToElastic[urlToParse]
				core.Units.Redis.Incr("running_tasks_cnt")
				fmt.Println("run task --")
				err := core.Units.Redis.Set("ruscraper_parse_lock", "1", 1).Err()
				if err != nil {
					fmt.Println("error on redis", err)
				}
				queue.RunTask(task)
			}
		});
	}

	router.Run()
}