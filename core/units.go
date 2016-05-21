package core

import (
	"fmt"
	"log"
	"time"
	"strings"
	"gopkg.in/redis.v3"
	"github.com/adjust/rmq"
	"gopkg.in/olivere/elastic.v3"	
	"ruscraper/conf"
	"ruscraper/models"
)

type FuncUnits struct {
	Redis *redis.Client
	Elastic *elastic.Client
	Queue rmq.Queue
} 

var Units = FuncUnits{}
var Config = conf.Conf{}

func SetupRedis() {

    Units.Redis = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
}

func SetupDb() {

	//SQLITE

	models.CheckAndCreateTable("filters", models.ThemeFilter{})
	models.CheckAndCreateTable("results", models.ParseResult{})

}

func SetupElastic() {

    client, err := elastic.NewClient()
	if err != nil {
    	// Handle error
	}

	fmt.Println("elastic - ok")
	Units.Elastic = client

	for _, indexName := range(Config.ElasticIndexes) {
		_, err = client.CreateIndex(indexName).Do()
		if err != nil {			
	    	// Handle error  
			if !strings.Contains(fmt.Sprintf("%s", err), "index_already_exists_exception") {
				log.Fatalf("elastic - CreateIndex", err)
	    		panic(err)
			}
		}
		fmt.Printf("index %s - ok\r\n", indexName)
	}
}

func SetupQueue() {
	connection := rmq.OpenConnection("redis", "tcp", "localhost:6379", 1)
	taskQueue := connection.OpenQueue("ParseTaskQueue")
	taskQueue.StartConsuming(10, time.Second)
	Units.Queue = taskQueue
}

func InitConfig() {

	Config.Read()
	SetupDb()
	SetupRedis()
	SetupElastic()
	SetupQueue()
}