package queue
//очередь для распределения запросов от шедулеров для постоянных запросов и фильтрации

import (
	"fmt"
	"time"
	"encoding/json"
	"github.com/adjust/rmq"
	"github.com/satori/go.uuid"
	"ruscraper/core"
	"ruscraper/models"
	"ruscraper/parser"
)

type ParseTask struct {
	NumPages int
	Url string
	Action string
	IndexName string
	Result []models.Theme
	Uuid string
}

type TaskConsumer struct {

}


func SetupConsumers() {

	taskStartConsumer := &TaskConsumer{}

	core.Units.Queue.AddConsumer("task consumer", taskStartConsumer)

}

type ParseTaskResult struct {
	Records []models.Theme
	Uuid string
}

var resultChan chan ParseTaskResult = make(chan ParseTaskResult)

func ParseResultToQueue() {

	for {
		result := <- resultChan
		fmt.Println("GET response", len(result.Records), result.Uuid)
		
		task := ParseTask{}
		task.Uuid = result.Uuid
		task.Result = result.Records
		task.Action = "end"
		taskBytes, _ := json.Marshal(task)

		if len(result.Records) > 0 {
			core.Units.Queue.PublishBytes(taskBytes)
			core.Units.Redis.Decr("running_tasks_cnt")
		}
	}
}

func RunTask(task ParseTask) {

    uuid_str := fmt.Sprintf("%s", uuid.NewV4())
    task.Uuid = uuid_str
	taskBytes, _ := json.Marshal(task)
	core.Units.Queue.PublishBytes(taskBytes)
}

func (self *TaskConsumer) Consume(delivery rmq.Delivery) {

	var task ParseTask
    if err := json.Unmarshal([]byte(delivery.Payload()), &task); err != nil {
        // handle error
        fmt.Println("bad task")
        delivery.Reject()
        return
    }

    if task.Action == "start" {    	
	    r1 := models.ParseResult{0, time.Now().Unix(), "parse", "pending", task.Uuid}
	    r1.SaveToDb()
	    fmt.Println("START parsing", task.Url)
	    go func() {
	    	res := parser.StartParse(task.Url, task.NumPages)
			resultChan <- ParseTaskResult{res, task.Uuid}
		}()
    } else if task.Action == "end" {
    	fmt.Println("END parsing", task.Url)
		r1 := models.ParseResult{0, time.Now().Unix(), "parse", "complete", task.Uuid}
	    r1.SaveToDb()
    }

    return
}
