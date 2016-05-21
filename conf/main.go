package conf

import (
	"os"
	"fmt"
	"log"
	"encoding/json"
)

type Conf struct {
	ParseUrls []string `json:"parse_urls"`
	ParsePagesNum int `json:"parse_pages_num"`
	ElasticIndexes []string `json:"elastic.indexes"`
	ParseInterval int `json:"parse_interval"`
	UrlToElastic map[string]string `json:"url_to_elastic"`
}

func (self *Conf) Read() (err error) {

	conf_file, err := os.Open("./conf/config.json")

	if err != nil {
		log.Fatalf("config empty")
		return err
	}

	decoder := json.NewDecoder(conf_file)

	err = decoder.Decode(self)

	if err != nil {
		log.Fatalf("Config is invalid!")
	}

	fmt.Printf("config=%s\r\n", self)

	return nil
}