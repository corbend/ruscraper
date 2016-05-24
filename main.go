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
	"github.com/gorilla/websocket"
)

var wsupgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

type WebsockMessage struct {
	Action string
	Payload []byte
}

type ParseStatusPayload struct {
	Url string
}

func wshandler(w http.ResponseWriter, r *http.Request) {
    conn, err := wsupgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("Failed to set websocket upgrade: %+v", err)
        return
    }

    for {
        t, msg, err := conn.ReadMessage()
        if err != nil {
            break
        }

        fmt.Println("web socket request", msg)

        var isActive = false;

        incomeWebsockMessage := WebsockMessage{}

        err = json.Unmarshal(msg, &incomeWebsockMessage)

        if incomeWebsockMessage.Action == "get_active_status" {
        	for i := 0; i < 10; i++ {
        		r, _ := core.Units.Redis.Get("active_parse_task_" + strconv.Itoa(i)).Result()
        		if r == string(incomeWebsockMessage.Payload) {
        			isActive = true
        			websockMessage := WebsockMessage{}
        			websockMessage.Action = "parse_active"
        			payload := ParseStatusPayload{}
        			payload.Url = string(incomeWebsockMessage.Payload)
        			p, _ := json.Marshal(&payload)
        			websockMessage.Payload = p
        			p, _ = json.Marshal(&websockMessage)
        			conn.WriteMessage(t, p)
        			conn.WriteMessage(t, msg)
        		}
        	}

        	if !isActive {
        		websockMessage := WebsockMessage{}
        		websockMessage.Action = "parse_nonactive"
        		payload := ParseStatusPayload{}
        		payload.Url = string(incomeWebsockMessage.Payload)
        		p, _ := json.Marshal(&payload)
        		websockMessage.Payload = p
        		p, _ = json.Marshal(&websockMessage)
        		conn.WriteMessage(t, p)
        	}
        }
    }
}

type ElasticIndexStat struct {
	Name string
	TotalDocs int64
}

func main() {

	core.InitConfig()

	router := gin.Default()

	router.Static("/assets", "./assets")
	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	router.GET("/ws", func(c *gin.Context) {
        wshandler(c.Writer, c.Request)
    })

	router.GET("/stat", func(c *gin.Context) {

		date := time.Now()
		year, month, day := date.Date()
		dateStr := fmt.Sprintf("%d-%d-%dT%d:00:00", year, month, day, date.Hour())

		parseAttemps, _ := core.Units.Redis.Get("parse_attemps").Result()
		runningTasksCnt, _ := core.Units.Redis.Get("running_tasks_cnt").Result()
		newHitsCnt1, _ := core.Units.Redis.Get("new_hits_cnt_" + dateStr + "_programming_books").Result()
		newHitsCnt2, _ := core.Units.Redis.Get("new_hits_cnt_" + dateStr + "_programming_videos").Result()

		lastUpdateTime, _ := core.Units.Redis.Get("new_hits_update_time").Result()

		indexes := []string{"programming_videos", "programming_books"}

		indexesStats := []ElasticIndexStat{}

		for _, idxName := range(indexes) {
			stats, _ := core.Units.Elastic.IndexStats(idxName).Do()
			stat := stats.Indices[idxName]

			indexesStats = append(indexesStats, ElasticIndexStat{
				idxName,
				stat.Total.Docs.Count,
			})
		}
		
		c.JSON(200, gin.H{
			"parse_attemps": parseAttemps,
			"new_hits_cnt_" + dateStr + "_programming_books": newHitsCnt1,
			"new_hits_cnt_" + dateStr + "_programming_videos": newHitsCnt2,
			"new_hits_update_time": lastUpdateTime,
			"running_tasks_cnt": runningTasksCnt,
			"redisStat": "{}",
			"elasticIndexesStats": indexesStats,
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

	router.GET("/categories", func(c *gin.Context) {
		c.JSON(200, models.GetAllCategories(c))
	})

	router.POST("/categories", func(c *gin.Context) {

		fmt.Println("save category")
		var newCategory models.ThemeCategory
		c.Bind(&newCategory)
		newCategory.SaveCategoryToDb()

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

		filterName, _ := c.Params.Get("filter_name")
		indexName := c.Request.URL.Query().Get("indexName")
		fmt.Println("INDEX", indexName)
		var records = []*models.Theme{}

		if filterName == "LastDay" {
			records, _ = models.GetLastThemes(core.Units.Elastic, indexName, 0, time.Second)
		} else if filterName == "Last5Days" {
			records, _ = models.GetLastThemes(core.Units.Elastic, indexName, 24 * 5, time.Hour)
		} else if filterName == "Last10Days" {
			records, _ = models.GetLastThemes(core.Units.Elastic, indexName, 24 * 10, time.Hour)
		} else if filterName == "LastMonth" {
			records, _ = models.GetLastThemes(core.Units.Elastic, indexName, 24 * 31, time.Hour)
		}
		
		results := gin.H{}

	 	for index, r := range(records) {
			results[strconv.Itoa(index)] = *r
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

	for w := 0; w < 10; w++ {
		go queue.ParseResultToQueue()	
	}
	
	scheduler.RunTimer(core.Config.ParseInterval, 	
		func() {
			r, _ := core.Units.Redis.Get("ruscraper_parse_lock").Result()
			for k, urlToParse := range(core.Config.ParseUrls) {
				if r == "" {					
					go func(urlToParse string) {
						task := queue.ParseTask{}
						task.Url = urlToParse
						task.Action = "start"
						task.NumPages = core.Config.ParsePagesNum
						task.IndexName = core.Config.UrlToElastic[urlToParse]
						core.Units.Redis.Incr("running_tasks_cnt")
						core.Units.Redis.Set("active_parse_task_" + strconv.Itoa(k), urlToParse, 0)
						err := core.Units.Redis.Set("ruscraper_parse_lock", "1", 1).Err()
						if err != nil {
							//fmt.Println("error on redis", err)
						}

						queue.RunTask(task)
					}(urlToParse)
					time.Sleep(120 * time.Millisecond)
				}
			}
		});

	router.Run()
}