package models

import (
	"time"
)

type ShortURL struct {
	ID          string    `json:"id" db:"id"`
	URL         string    `json:"url" db:"url" validate:"required,url"`
	ShortCode   string    `json:"shortCode" db:"short_code"`
	AccessCount int64     `json:"accessCount" db:"access_count"`
	Created     time.Time `json:"created" db:"created"`
	Updated     time.Time `json:"updated" db:"updated"`
}

type PBShortURL struct {
	ID             string    `json:"id"`
	CollectionId   string    `json:"collectionId"`
	CollectionName string    `json:"collectionName"`
	Created        time.Time `json:"created"`
	Updated        time.Time `json:"updated"`
	URL            string    `json:"url"`
	ShortCode      string    `json:"short_code"`
	AccessCount    int64     `json:"access_count"`
}

type CreateShortURLRequest struct {
	URL string `json:"url" binding:"required,url"`
}

type UpdateShortURLRequest struct {
	URL string `json:"url" binding:"required,url"`
}

type ShortURLResponse struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	ShortCode   string    `json:"shortCode"`
	AccessCount int64     `json:"accessCount,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (su *ShortURL) ToResponse() *ShortURLResponse {
	return &ShortURLResponse{
		ID:          su.ID,
		URL:         su.URL,
		ShortCode:   su.ShortCode,
		AccessCount: su.AccessCount,
		CreatedAt:   su.Created,
		UpdatedAt:   su.Updated,
	}
}

func (su *ShortURL) FromPBRecord(pb *PBShortURL) {
	su.ID = pb.ID
	su.URL = pb.URL
	su.ShortCode = pb.ShortCode
	su.AccessCount = pb.AccessCount
	su.Created = pb.Created
	su.Updated = pb.Updated
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
