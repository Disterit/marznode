package models

type Inbound struct {
	Tag      string
	Protocol string
	Config   map[string]interface{}
}
