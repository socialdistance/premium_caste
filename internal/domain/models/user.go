package models

import "time"

type User struct {
	ID                int64     `db:"id" json:"id"`
	Name              string    `db:"name" json:"name"`
	Email             string    `db:"email" json:"email"`
	Phone             string    `db:"phone" json:"phone"`
	Password          []byte    `db:"password" json:"password"`
	Permission_id     int       `db:"permission_id" json:"permission_id"`
	Basket_id         int       `db:"basket_id" json:"basket_id"`
	Registration_date time.Time `db:"registration_date,omitempty" json:"registration_date,omitempty"`
	Last_login        time.Time `db:"last_login,omitempty" json:"last_login,omitempty"`
}
