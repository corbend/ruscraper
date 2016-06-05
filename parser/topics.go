package parser

import (
	"fmt"
	"strings"
	"strconv"
	"regexp"
	"github.com/PuerkitoBio/goquery"
	"ruscraper/core"
	"ruscraper/models"
)

func ParseTopics() []models.Topic {

	var doc *goquery.Document
	baseUrl := "http://maintracker.org/forum/"
	url := baseUrl + "/index.php"
	doc, _ = goquery.NewDocument(url)

	topics := []models.Topic{}

	if doc == nil {
		return nil
	}

	cnt := 0
	linkUrls := []string{} 

	if doc != nil {
		doc.Find(".forumlink a").Each(func(i int, s *goquery.Selection) {

			href, _ := s.Attr("href")
			fmt.Println("archi topics", href)

			linkUrls = append(linkUrls, baseUrl + href)		
		})
	}

	topicIdReg, _ := regexp.Compile("\\d+")

	for _, lurl := range(linkUrls) {
		subdoc, _ := goquery.NewDocument(lurl)

		if subdoc != nil {
			subdoc.Find(".forumlink a").Each(func(i int, s *goquery.Selection) {

				title := s.Text()
				decodedTitle := strings.Replace(FromCharmap(title), "\r\n", "", -1)
				t := models.Topic{}
	 			t.Title = decodedTitle
	 			href, _ := s.Attr("href")
	 			t.ParseUrl = href
	 			tid, _ := strconv.Atoi(topicIdReg.FindString(href))
	 			t.ExternalId = tid
	 			fmt.Println("get topic", t, tid)			
				topics = append(topics, t)
				cnt += 1	
			})
		}

		if cnt > 200 {
			break
		}
	}

	return topics
}

func GetAllTopics() {

	predefinedIndexes := ParseTopics()
	fmt.Println("groups", predefinedIndexes)
	indexes := []int{}

	for _, i := range(predefinedIndexes) {
		indexes = append(indexes, i.ExternalId)
	}

	bucketsCnt := len(indexes) / 100
	existedTopics := map[int]int{}

	for i := 0; i < bucketsCnt; i++ {
		for k, v := range(models.FindTopicIds(indexes[i*100:i*100+100])) {
			existedTopics[k] = v
		}
	}

	for k, v := range(models.FindTopicIds(indexes[bucketsCnt*100:bucketsCnt*100+len(indexes) % 100])) {
		existedTopics[k] = v
	}

	fmt.Println(existedTopics)

	mapTopicToBlackList := map[int]bool{}

	for _, extId := range(core.Config.TopicBlackList) {
		mapTopicToBlackList[extId] = true
	}

	for _, t := range(predefinedIndexes) {

		if existedTopics[t.ExternalId] == 0 {
			if mapTopicToBlackList[t.ExternalId] != false {
				t.Restrict = true
			}
			models.CreateTopic(t)
		}
	}
}