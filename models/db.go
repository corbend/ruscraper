package models

import (
	"fmt"
	"log"
	"reflect"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

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

	fmt.Printf("query execution OK\r\n")

	return db, rows
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

func CreateTable(tableName string, model interface{}) (bool, error) {
	fmt.Println("CREATE TABLE")

	tableColumns := ""

 	v := reflect.ValueOf(model)

	for i := 0; i < v.NumField(); i++ {

		value := v.Field(i).Interface()
		tagName := v.Type().Field(i).Tag.Get("json")
		fieldName := v.Type().Field(i).Name

		_, is_string := value.(string)
		_, is_int := value.(int)
		_, is_int64 := value.(int64)
		_, is_array := value.([]interface{})
		_, is_str_array := value.([]string)
		_, is_int_array := value.([]int)

		if fieldName != "Id" {
			if is_string {
				tableColumns += fmt.Sprintf(", %s VARCHAR", tagName)
			} else if is_int64 || is_int {
				tableColumns += fmt.Sprintf(", %s INTEGER", tagName)
			} else if is_array || is_str_array || is_int_array {
				fmt.Println("skip field", fieldName)
			} else {
				log.Fatalf("not recognized field type", fieldName)
			}
		}
	}

	query := fmt.Sprintf("CREATE TABLE %s(id INTEGER PRIMARY KEY AUTOINCREMENT%s)", tableName, tableColumns);
	fmt.Println(query)
	db2, r2 := RunQuery(query)

	var created string
	r2.Next()
	r2.Scan(&created)

	fmt.Println("CREATE TABLE OK")

	defer db2.Close()
	defer r2.Close()

	return created != "", nil
}

func CheckAndCreateTable(tableName string, model interface{}) (bool) {

	res, err := CheckTable(tableName)

	if err != nil {
		log.Fatalf("create table error")
	}

	if !res {
		res, err = CreateTable(tableName, model)

		if err != nil {
			log.Fatalf("create table error")
		}
	}

	return res

}
