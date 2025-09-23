package dto

import "time"

type CreateURLRequest struct {
	URL        string  `json:"url" validate:"required,url,max=2048"`
	CustomCode *string `json:"customCode,omitempty" validate:"omitempty,min=4,max=20,alphanum"`
}

type CreateURLResponse struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	ShortCode string    `json:"shortCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GetURLResponse struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	ShortCode   string    `json:"shortCode"`
	AccessCount int64     `json:"accessCount,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UpdateURLRequest struct {
	URL string `json:"url" validate:"required,url,max=2048"`
}

type UpdateURLResponse struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	ShortCode   string    `json:"shortCode"`
	AccessCount int64     `json:"accessCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type GetStatsResponse struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	ShortCode   string    `json:"shortCode"`
	AccessCount int64     `json:"accessCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}
