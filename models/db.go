package models

import (
	"fmt"
	"log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type ThemeFilter struct {
	TermName string `json:"term_name"`
	TermValues string `json:"term_values"`
	TermValuesList []string
	FilterType string `json:"filter_type"`
	ElasticFilter string `json:"elastic_filter"`
}

func ConnectToDb(dbName string) (db *sql.DB) {
	db, err := sql.Open("sqlite3", dbName)

	if err != nil {
		log.Fatalf("error on connection to database %s", dbName)
	}

	return db
}

const (
	DB_NAME="settings.db"
)

func RunQuery(query string) (db *sql.DB, rows *sql.Rows){

	db = ConnectToDb(DB_NAME);

	rows, err := db.Query(query)

	if err != nil {
		log.Fatalf("errors on query %s\r\n", err)
	}

	fmt.Printf("query execution OK %s\r\n")

	return db, rows
}

func (self *ThemeFilter) SaveToDb() (bool, error) {

	query := "INSERT INTO filters (type, name, fvalues, elastic_filter) VALUES ("
	query += fmt.Sprintf("'%s'", self.FilterType) + ","		
	query += fmt.Sprintf("'%s'", self.TermName) + ","
	query += fmt.Sprintf("'%s'", self.TermValues) + ","
	query += fmt.Sprintf("'%s'", self.ElasticFilter) + ");"
	
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

func CheckTable(tableName string) (bool, error) {

	query := fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'", tableName);
	db2, r2 := RunQuery(query)

	var exist string
	r2.Next()
	r2.Scan(&exist)

	defer db2.Close()
	defer r2.Close()

	return exist != "", nil
}

func CreateTable(tableName string) (bool, error) {
	fmt.Println("CREATE TABLE")
	query := fmt.Sprintf("CREATE TABLE %s(id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR, type VARCHAR, fvalues VARCHAR, elastic_filter VARCHAR)", tableName);
	db2, r2 := RunQuery(query)

	var created string
	r2.Next()
	r2.Scan(&created)

	fmt.Println("CREATE TABLE OK")

	defer db2.Close()
	defer r2.Close()

	return created != "", nil
}

func CheckAndCreateTable(tableName string) (bool) {

	res, err := CheckTable(tableName)

	if err != nil {
		log.Fatalf("create table error")
	}

	if !res {
		res, err = CreateTable(tableName)

		if err != nil {
			log.Fatalf("create table error")
		}
	}

	return res

}