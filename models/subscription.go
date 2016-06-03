package models

import (
	"fmt"
)

type ForeignKey struct {
	FromField string
	ToField string
	RefTableName string
}

type SubscriptionForm struct {
	Categories []int `json:"categories"`
}

type Subscription struct {
	Id int `json:"id"`
	UserId int `json:"user_id"`
	CategoryId int `json:"category_id"`
	CategoryName string `json:"name"`
	IndexName string `json:"index_name"`
	fk1 ForeignKey `fk:"user_id,id,users"`
	fk2 ForeignKey `fk:"category_id,id,categories"`
}

func GetAllSubscriptions(userId int) ([]Subscription, error) {

	query := "SELECT id, category_id FROM subscriptions WHERE user_id=%d"

	query = fmt.Sprintf(query, userId)

	db2, r2 := RunQuery(query)

	result := []Subscription{}

	for r2.Next() {
		subs := Subscription{}
		r2.Scan(&subs.Id, &subs.CategoryId)
		result = append(result, subs)
	}
	
	defer db2.Close()
	defer r2.Close()

	return result, nil
}

func DeleteSubscriptions(userId int) (bool, error) {

	query := "DELETE FROM subscriptions WHERE user_id=%d"

	query = fmt.Sprintf(query, userId)

	db2, r2 := RunQuery(query)
	
	defer db2.Close()
	defer r2.Close()

	return true, nil
}

func GetAllSubscriptionsJoined(userId int) ([]Subscription, error) {

	query := "SELECT s.id, s.category_id, c.name FROM subscriptions s LEFT JOIN categories c ON s.category_id = c.id WHERE s.user_id=%d"

	query = fmt.Sprintf(query, userId)

	db2, r2 := RunQuery(query)

	result := []Subscription{}

	for r2.Next() {
		subs := Subscription{}
		r2.Scan(&subs.Id, &subs.CategoryId, &subs.CategoryName)
		result = append(result, subs)
	}
	
	defer db2.Close()
	defer r2.Close()

	return result, nil
}

func (self *Subscription) SaveSubscriptionToDb() (bool, error) {

	query := "INSERT INTO subscriptions (user_id, category_id) VALUES ("
	query += fmt.Sprintf("%d", self.UserId) + ","		
	query += fmt.Sprintf("%d", self.CategoryId) + ");"
	
	db2, r2 := RunQuery(query)

	var created string
	r2.Next()
	r2.Scan(&created)

	defer db2.Close()
	defer r2.Close()

	return true, nil
}