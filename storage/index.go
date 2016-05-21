package storage

import (
	"fmt"
	"time"
	"strconv"
	"gopkg.in/olivere/elastic.v3"
	"ruscraper/models"
	"ruscraper/core"
)

func SaveToStore(themes []models.Theme, indexName string) {
	
	//add themes to elastic search
	for _, t := range(themes) {

		termQuery := elastic.NewTermQuery("Id", t.Id)

		searchResult, err := core.Units.Elastic.Search().
		    Index(indexName).
		    Query(termQuery).   // specify the query
		    Sort("Name", true). // sort by "user" field, ascending
		    From(0).Size(1).   // take documents 0-9
		    Pretty(true).       // pretty print request and response JSON
		    Do()                // execute

		if err != nil {
		    // Handle error
		    fmt.Println("elastic - search fail", err)
		    // panic(err)
		}

		fmt.Println("elastic - search theme", t.Name, searchResult.TotalHits())

		t.Read = true

		if searchResult == nil || searchResult.TotalHits() == int64(0) {

			t.CreateDate = time.Now().Unix()
			
			t_id := strconv.Itoa(int(t.Id))
			_, err = core.Units.Elastic.Index().
			    Index(indexName).
			    Type("theme").
			    Id(t_id).
			    BodyJson(t).
			    Do()

			if err != nil {
			    // Handle error
			    panic(err)
			}
			date := time.Now()
			year, month, day := date.Date()
			dayHourNewHitsCnt := fmt.Sprintf("%d-%d-%dT%d:00:00", year, month, day, date.Hour())
			lastUpdateTime := date.Format(time.RFC3339)
			_ = core.Units.Redis.Incr("new_hits_cnt_" + dayHourNewHitsCnt + "_" + indexName).Err()
			_ = core.Units.Redis.Set("new_hits_update_time", lastUpdateTime, 0).Err()
			t.Read = false
		} else {
			fmt.Println("skip")
		}
	}
}