package models

import (
	"fmt"
	"strings"
	"strconv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/gin-gonic/gin"
)

type ThemeFilter struct {
	Id int `json:"id"`
	TermName string `json:"term_name"`
	TermValues string `json:"term_values"`
	TermValuesList []string
	FilterType string `json:"filter_type"`
	ElasticFilter string `json:"elastic_filter"`
	IndexName string `json:"index_name"`
}

func (self *ThemeFilter) SaveToDb() (bool, error) {

	query := "INSERT INTO filters (filter_type, term_name, term_values, elastic_filter, index_name) VALUES ("
	query += fmt.Sprintf("'%s'", self.FilterType) + ","		
	query += fmt.Sprintf("'%s'", self.TermName) + ","
	query += fmt.Sprintf("'%s'", self.TermValues) + ","
	query += fmt.Sprintf("'%s'", self.ElasticFilter) + ","
	query += fmt.Sprintf("'%s'", self.IndexName) + ");"
	
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


func GetAllFilters(c *gin.Context) (results gin.H) {
	fmt.Println("get filters")

	filtersQuery := "SELECT id, filter_type, term_name, term_values, elastic_filter, index_name FROM filters"

	db1, r1 := RunQuery(filtersQuery)

	results = gin.H{}
	cnt := 0
	for r1.Next() {
		var themeFilter ThemeFilter
		termValues := ""

		r1.Scan(&themeFilter.Id, &themeFilter.FilterType, &themeFilter.TermName, &themeFilter.TermValues, &themeFilter.ElasticFilter, &themeFilter.IndexName)
		themeFilter.TermValuesList = strings.Split(termValues, ",")
		results[strconv.Itoa(cnt)] = themeFilter
		cnt += 1
	}

	r1.Close()
	db1.Close()

	return gin.H{"filters": results}
}

func RemoveFilter(filterId int, c *gin.Context) (results gin.H) {


	filtersQuery := fmt.Sprintf("DELETE FROM filters WHERE id = %d", filterId)

	db1, r1 := RunQuery(filtersQuery)

	r1.Close()
	db1.Close()

	fmt.Println("destroy filter", filtersQuery, filterId)

	return gin.H{"success": true}
}