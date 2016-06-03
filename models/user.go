package models

import (
	"fmt"
)

type User struct {
	Id int `json:"id"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Password string `json:"password"`
	Login string `json:"login"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (self *User) GetFromDb() (int, error) {

	query := "SELECT id FROM users WHERE email='%s' AND password='%s'"

	query = fmt.Sprintf(query, self.Email, self.Password)

	db2, r2 := RunQuery(query)

	r2.Next()
	r2.Scan(&self.Id)

	fmt.Println("GET USER", self)

	defer db2.Close()
	defer r2.Close()

	return self.Id, nil
}

func (self *User) SaveUserToDb() (bool, error) {

	query := "INSERT INTO users (first_name, last_name, password, login, email, phone) VALUES ("
	query += fmt.Sprintf("'%s'", self.FirstName) + ","		
	query += fmt.Sprintf("'%s'", self.LastName) + ","
	query += fmt.Sprintf("'%s'", self.Password) + ","
	query += fmt.Sprintf("'%s'", self.Login) + ","
	query += fmt.Sprintf("'%s'", self.Email) + ","
	query += fmt.Sprintf("'%s'", self.Phone) + ");"
	
	db2, r2 := RunQuery(query)

	var created string
	r2.Next()
	r2.Scan(&created)

	defer db2.Close()
	defer r2.Close()

	return true, nil
}