package models

import "time"

type User struct {
	ID                int
	Name              string
	Email             string
	Phone             string
	Password          string
	Permission_id     int
	Basket_id         int
	Registration_date time.Time
	Last_login        time.Time
}
