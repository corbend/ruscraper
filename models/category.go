package models

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type ThemeCategory struct {
	Id int `json:"id"`
	Name string `json:"name"`
}

func (self *ThemeCategory) SaveCategoryToDb() (bool, error) {

	query := "INSERT INTO categories (name) VALUES ("
	query += fmt.Sprintf("'%s'", self.Name) + ");"

	db2, r2 := RunQuery(query)

	var created string
	r2.Next()
	r2.Scan(&created)

	defer db2.Close()
	defer r2.Close()

	return true, nil
}

func GetAllCategories() ([]ThemeCategory) {
	fmt.Println("get categories")

	filtersQuery := "SELECT id, name FROM categories"

	db1, r1 := RunQuery(filtersQuery)

	results := []ThemeCategory{}
	cnt := 0
	for r1.Next() {
		var themeFilter ThemeCategory
		r1.Scan(&themeFilter.Id, &themeFilter.Name)
		results = append(results, themeFilter)
		cnt += 1
	}

	r1.Close()
	db1.Close()

	return results
}
