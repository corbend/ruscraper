package main

import (
	"os"
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
	"ruscraper/storage"
	"ruscraper/models"
	"ruscraper/helpers"
	"gopkg.in/olivere/elastic.v3"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsupgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

type WebsockMessage struct {
	Action string `json:"action"`
	Payload interface{} `json:"payload"`
}

type ParseStatusPayload struct {
	Url string
}

type LastUpdatePayload struct {
	Items []models.SearchTheme
}

type GetUpdateInPayload struct {
	UserId int `json:"user_id"`
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

        var isActive = false;

        incomeWebsockMessage := WebsockMessage{}

        err = json.Unmarshal(msg, &incomeWebsockMessage)

        if err != nil {
        	fmt.Println("web socket message deserialization - error is occured")
        }

        fmt.Println("web socket request", t, incomeWebsockMessage)

        if incomeWebsockMessage.Action == "get_active_status" {
        	for i := 0; i < 10; i++ {
        		r, _ := core.Units.Redis.Get("active_parse_task_" + strconv.Itoa(i)).Result()
        		if r == incomeWebsockMessage.Payload.(string) {
        			isActive = true
        			websockMessage := WebsockMessage{}
        			websockMessage.Action = "parse_active"
        			payload := ParseStatusPayload{}
        			payload.Url = incomeWebsockMessage.Payload.(string)
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
        		payload.Url = incomeWebsockMessage.Payload.(string)
        		p, _ := json.Marshal(&payload)
        		websockMessage.Payload = p
        		p, _ = json.Marshal(&websockMessage)
        		conn.WriteMessage(t, p)
        	}
        } else if incomeWebsockMessage.Action == "get_updates" {
        		
    		inPayloadMap := incomeWebsockMessage.Payload.(map[string]interface{})

    		userIdStr := inPayloadMap["user_id"].(string)

    		if err != nil {
    			fmt.Println("error on Deserialization")
    		}

        	fmt.Println("GET UPDATE ---", incomeWebsockMessage, userIdStr)
        	websockMessage := WebsockMessage{}
        	websockMessage.Action = "get_updates"
        	
        	payload := LastUpdatePayload{}
        	userIdInt, _ := strconv.Atoi(userIdStr)
        	subscriptions, _ := models.GetAllSubscriptionsJoined(userIdInt)

        	themes := []models.SearchTheme{}
        	themeIds := map[int64]string{}

        	for _, indexName := range([]string{"programming_videos", "programming_books"}) {
        		for _, subs := range(subscriptions) {
        			items, _ := storage.GetLastItems(subs.CategoryName, indexName)
        			for _, i := range(items) {
        				ii := models.SearchTheme{}
        				ii.Id = i.Id
        				ii.Name = i.Name
        				ii.CreateDate = i.CreateDate
        				ii.PubYear = i.PubYear
        				ii.Size = i.Size
						ii.Date = i.Date
						ii.Answers = i.Answers
						ii.SearchTerms = []string{subs.CategoryName}

        				if themeIds[i.Id] == "" {
        					themes = append(themes, ii)
        					themeIds[i.Id] = i.Name
        				}
        			}					
				}
        	}

        	payload.Items = themes
    		websockMessage.Payload = payload
    		p, _ := json.Marshal(&websockMessage)
        	conn.WriteMessage(t, p)
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
		parseFails, _ := helpers.GetTimeCounter("parse_fails")
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
			"parse_fail": parseFails,
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
		c.JSON(200, gin.H{"rows": models.GetAllCategories()})
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

		var termQuery elastic.Query

		if applyFilter.FilterType == "exact" {
			termQuery = elastic.NewTermQuery(applyFilter.TermName, applyFilter.TermValues)
		} else if applyFilter.FilterType == "match" {
			termQuery = elastic.NewMatchQuery(applyFilter.TermName, applyFilter.TermValues)
		} else {
			fmt.Println("like query", applyFilter)
			termQuery = elastic.NewMoreLikeThisQuery().LikeText(applyFilter.TermValues).Field(applyFilter.TermName)
		}

		searchResult, err := core.Units.Elastic.Search().
		    Index(applyFilter.IndexName).
		    Query(termQuery).
		    Sort("CreateDate", true). 
		    From(0).Size(1000).
		    Pretty(true).
		    Do()

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

	router.GET("/topics", func(c *gin.Context) {
		c.JSON(200, models.GetAllTopics())	
	})

	router.POST("/users", func(c *gin.Context) {
		newUser := models.User{}
		c.Bind(&newUser)

		_, err := newUser.SaveUserToDb()

		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
			})
		} else {
			c.JSON(200, gin.H{
				"success": true,
			})
		}
	})

	router.POST("/login", func(c *gin.Context) {
		newUser := models.User{}
		c.Bind(&newUser)

		userId, _ := newUser.GetFromDb()

		if userId > 0 {
			c.JSON(200, gin.H{"success": true, "Id": userId})
		} else {
			fmt.Println("user not found")
			c.JSON(404, gin.H{"success": false, "message": "user not found"})
		}
	})

	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})

	router.GET("/users/register/", func(c *gin.Context) {		
		c.HTML(http.StatusOK, "register.html", gin.H{})	
	})

	router.GET("/private/:userId", func(c *gin.Context) {
		//TODO - проверка сессии
		c.HTML(http.StatusOK, "user-dashboard.html", gin.H{})	
	})

	router.POST("/users/:id/subscribe", func(c *gin.Context) {

		userId, _ := c.Params.Get("id")
		userIdInt, _ := strconv.Atoi(userId)

		var form = models.SubscriptionForm{}

		c.Bind(&form)

		models.DeleteSubscriptions(userIdInt)

		for _, t := range(form.Categories) {
			core.Units.Redis.SAdd("user_" + userId, strconv.Itoa(t))
			subs := models.Subscription{}

			subs.UserId = userIdInt
			subs.CategoryId = t
			subs.SaveSubscriptionToDb()
		}

		c.JSON(200, gin.H{"success": true})
	})

	router.GET("/subscriptions/:userId", func(c *gin.Context) {

		userId, _ := c.Params.Get("userId")
		userIdInt, _ := strconv.Atoi(userId)
		subscriptions, _ := models.GetAllSubscriptions(userIdInt)

		c.JSON(200, gin.H{"rows": subscriptions})
	})

	router.GET("/subscriptions/:userId/counters", func(c *gin.Context) {

		c.JSON(200, gin.H{"counters": []int{}})
	})

	//get all topics from forum and save to model
	if len(os.Args) == 2 && os.Args[1] == "hot" {
		parser.GetAllTopics()
	}

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