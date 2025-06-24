package models

type Inbound struct {
	Tag      string         `json:"tag" validate:"required"`
	Protocol string         `json:"protocol" validate:"required"`
	Config   map[string]any `json:"config" validate:"required"`
}
