package models

import "time"

type User struct {
	ID                int64
	Name              string
	Email             string
	Phone             string
	Password          []byte
	Permission_id     int
	Basket_id         int
	Registration_date time.Time
	Last_login        time.Time
}
