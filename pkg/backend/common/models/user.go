package models

type User struct {
	ID       int64     `json:"id" validate:"required"`
	Username string    `json:"username" validate:"required"`
	Key      string    `json:"key" validate:"required"`
	Inbounds []Inbound `json:"inbounds"`
}
