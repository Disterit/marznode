package models

type User struct {
	Id       int       `json:"id" binding:"required"`
	Username string    `json:"username" binding:"required"`
	Key      string    `json:"key" binding:"required"`
	Inbounds []Inbound `json:"inbounds"`
}
