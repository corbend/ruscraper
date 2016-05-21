package models

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type ParseResult struct {
	Id int `json:"id"`
	Date int64 `json:"date"`
	Type string `json:"type"`
	Status string `json:"status"`
	Uuid string `json:"uuid"`
}

func (self *ParseResult) SaveToDb() (bool, error) {

	query := "INSERT INTO results (type, date, status, uuid) VALUES ("
	query += fmt.Sprintf("'%s'", self.Type) + ","		
	query += fmt.Sprintf("%d", self.Date) + ","
	query += fmt.Sprintf("'%s'", self.Status) + ","
	query += fmt.Sprintf("'%s'", self.Uuid) + ");"
	
	fmt.Printf("run query -->\r\n%s\r\n, %s\r\n", query, self)
	db2, r2 := RunQuery(query)

	var created string
	r2.Next()
	r2.Scan(&created)

	fmt.Printf("create - %s\r\n", created)

	defer db2.Close()
	defer r2.Close()

	return true, nil
}
