package models

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	_ "github.com/mattn/go-sqlite3"
)

type Topic struct {
	Id int `json:"id"`
	Title string `json:"title"`
	IndexName string `json:"index_name"`
	ParseUrl string `json:"parse_url"`
	ExternalId int `json:"external_id"`
	Restrict bool `json:"restrict"`
}

func FindTopicIds(topicIds []int) map[int]int {

	strIds := []string{}

	for _, t := range(topicIds) {
		strIds = append(strIds, strconv.Itoa(t))
	}

	query := fmt.Sprintf("SELECT id, external_id, parse_url FROM topics WHERE external_id IN (%s)", strings.Join(strIds, ","))

	fmt.Println("topics query", query)

	results := []Topic{}

	db1, r1 := RunQuery(query)

	for r1.Next() {
		var topic Topic = Topic{}
		r1.Scan(&topic.Id, &topic.ExternalId, &topic.ParseUrl)
		results = append(results, topic)
	}

	r1.Close()
	db1.Close()

	mapIdToUrl := map[int]int{}	

	for _, r := range(results) {

		mapIdToUrl[r.ExternalId] = r.Id
	}

	return mapIdToUrl
}

func CreateTopic(topic Topic) (bool, error) {

	query := "INSERT INTO topics (title, external_id, restrict) VALUES ("
	query += fmt.Sprintf("'%s'", html.EscapeString(topic.Title)) + ","
	query += fmt.Sprintf("%d", topic.ExternalId) + ","

	if topic.Restrict {
		query +=  "1);"
	} else {
		query +=  "0);"
	}

	db2, r2 := RunQuery(query)

	r2.Next()

	defer db2.Close()
	defer r2.Close()

	return true, nil
}

func GetAllTopics() (results []Topic) {
	fmt.Println("get filters")

	query := "SELECT id, title, index_name FROM topics WHERE restrict=0 OR restrict IS NULL"

	db1, r1 := RunQuery(query)

	results = []Topic{}

	for r1.Next() {
		var topic Topic
		r1.Scan(&topic.Id, &topic.Title, &topic.IndexName)
		results = append(results, topic)
	}

	r1.Close()
	db1.Close()

	return results
}