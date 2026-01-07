package api

import "time"

// Tag represents a Docker Hub image tag
type Tag struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"last_updated"`
	FullSize    int64     `json:"full_size"`
	Images      []Image   `json:"images"`
}

// Image represents individual image layers in a tag
type Image struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Size         int64  `json:"size"`
}

// LoginRequest represents the Docker Hub login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the Docker Hub login response
type LoginResponse struct {
	Token string `json:"token"`
}

// TagsResponse represents the paginated tags response from Docker Hub
type TagsResponse struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []Tag   `json:"results"`
}

// Repository represents a Docker Hub repository
type Repository struct {
	User        string `json:"user"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
}
