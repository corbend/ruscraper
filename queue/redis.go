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

type TaskStartConsumer struct {

}

type TaskEndConsumer struct {

}

func SetupConsumers() {

	taskStartConsumer := &TaskStartConsumer{}
	taskEndConsumer := &TaskEndConsumer{}

	core.Units.Queue.AddConsumer("task start consumer", taskStartConsumer)
	core.Units.Queue.AddConsumer("task end consumer", taskEndConsumer)

}

var resultChan chan []models.Theme = make(chan []models.Theme)


func RunTask(task ParseTask) {

    uuid_str := fmt.Sprintf("%s", uuid.NewV4())
    task.Uuid = uuid_str
	taskBytes, _ := json.Marshal(task)
	core.Units.Queue.PublishBytes(taskBytes)

	go func() {
		result := <- resultChan

		fmt.Println("parse complete", len(result), uuid_str, task.Uuid)

		task.Result = result
		task.Action = "end"
		taskBytes, _ = json.Marshal(task)

		if len(result) > 0 {
			core.Units.Queue.PublishBytes(taskBytes)
			core.Units.Redis.Decr("running_tasks_cnt")
		}
	}()
}

func (self *TaskStartConsumer) Consume(delivery rmq.Delivery) {

	var task ParseTask
    if err := json.Unmarshal([]byte(delivery.Payload()), &task); err != nil {
        // handle error
        fmt.Println("bad task")
        delivery.Reject()
        return
    }

    if task.Action != "start" {
    	delivery.Reject()
    	return 
    }

    r1 := models.ParseResult{0, time.Now().Unix(), "parse", "pending", task.Uuid}
    r1.SaveToDb()

    resultChan <- parser.StartParse(task.Url, task.NumPages)
}

func (self *TaskEndConsumer) Consume(delivery rmq.Delivery) {
	
	var task ParseTask
    if err := json.Unmarshal([]byte(delivery.Payload()), &task); err != nil {
        // handle error
        fmt.Println("bad task")
        delivery.Reject()
        return
    }

	if task.Action != "end" {
		delivery.Reject()
		return 
	}

	//TODO - send by websocket

	r1 := models.ParseResult{0, time.Now().Unix(), "parse", "complete", task.Uuid}
    r1.SaveToDb()
}