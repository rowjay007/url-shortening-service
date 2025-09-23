package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rowjay/url-shortening-service/internal/constants"
	"github.com/rowjay/url-shortening-service/internal/database"
	serviceErrors "github.com/rowjay/url-shortening-service/internal/errors"
	urlModels "github.com/rowjay/url-shortening-service/internal/models"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type URLRepository interface {
	Create(ctx context.Context, shortURL *urlModels.ShortURL) error
	GetByShortCode(ctx context.Context, shortCode string) (*urlModels.ShortURL, error)
	Update(ctx context.Context, shortCode string, url string) (*urlModels.ShortURL, error)
	Delete(ctx context.Context, shortCode string) error
	IncrementAccessCount(ctx context.Context, shortCode string) error
	ExistsByShortCode(ctx context.Context, shortCode string) (bool, error)
}

type pocketBaseRecord struct {
	ID          string `json:"id"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	URL         string `json:"url"`
	ShortCode   string `json:"short_code"`
	AccessCount int64  `json:"access_count"`
}

type pocketBaseListResponse struct {
	Items []pocketBaseRecord `json:"items"`
}

type pocketBaseCreateRequest struct {
	URL         string `json:"url"`
	ShortCode   string `json:"short_code"`
	AccessCount int64  `json:"access_count"`
}

type pocketBaseUpdateRequest struct {
	AccessCount *int64  `json:"access_count,omitempty"`
	URL         *string `json:"url,omitempty"`
}

type urlRepositoryImpl struct {
	pb *database.PBClient
}

func parsePBTime(pbTime string) time.Time {
	if pbTime == "" {
		return time.Time{}
	}

	rfc3339Time := strings.Replace(pbTime, " ", "T", 1)

	t, err := time.Parse(time.RFC3339, rfc3339Time)
	if err != nil {
		log.Warn().Err(err).Str("time", pbTime).Msg("Failed to parse PocketBase timestamp")
		return time.Time{}
	}
	return t
}

func NewURLRepository(pb *database.PBClient) URLRepository {
	return &urlRepositoryImpl{pb: pb}
}

func (r *urlRepositoryImpl) Create(ctx context.Context, shortURL *urlModels.ShortURL) error {
	log.Debug().Str("short_code", shortURL.ShortCode).Str("url", shortURL.URL).Msg("Creating new short URL")

	reqBody := pocketBaseCreateRequest{
		URL:         shortURL.URL,
		ShortCode:   shortURL.ShortCode,
		AccessCount: 0,
	}

	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
	defer cancel()

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal create request")
		return serviceErrors.NewInternalError("repository.Create", "failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		r.pb.BaseURL+"/api/collections/"+constants.ShortURLsCollection+"/records",
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return serviceErrors.NewInternalError("repository.Create", "failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.pb.HTTPClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create short URL")
		return serviceErrors.NewInternalError("repository.Create", "failed to create record", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Error().Int("status", resp.StatusCode).Msg("PocketBase returned error status")
		if resp.StatusCode == http.StatusConflict {
			return serviceErrors.NewDuplicateError("repository.Create", "short code already exists")
		}
		return serviceErrors.NewInternalError("repository.Create", "PocketBase error", fmt.Errorf("status %d", resp.StatusCode))
	}

	var pbResp pocketBaseRecord
	if err := json.NewDecoder(resp.Body).Decode(&pbResp); err != nil {
		log.Error().Err(err).Msg("Failed to decode response")
		return serviceErrors.NewInternalError("repository.Create", "failed to decode response", err)
	}

	shortURL.ID = pbResp.ID
	shortURL.Created = parsePBTime(pbResp.Created)
	shortURL.Updated = parsePBTime(pbResp.Updated)

	log.Info().Str("short_code", shortURL.ShortCode).Str("id", pbResp.ID).Msg("Short URL record created successfully")
	return nil
}

func (r *urlRepositoryImpl) GetByShortCode(ctx context.Context, shortCode string) (*urlModels.ShortURL, error) {
	log.Debug().Str("short_code", shortCode).Msg("Looking up short URL by code")

	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
	defer cancel()

	filter := fmt.Sprintf("(short_code=\"%s\")", shortCode)
	reqURL := fmt.Sprintf("%s/api/collections/%s/records?filter=%s",
		r.pb.BaseURL, constants.ShortURLsCollection, url.QueryEscape(filter))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, serviceErrors.NewInternalError("repository.GetByShortCode", "failed to create request", err)
	}

	resp, err := r.pb.HTTPClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("short_code", shortCode).Msg("Failed to lookup short URL")
		return nil, serviceErrors.NewInternalError("repository.GetByShortCode", "failed to lookup record", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debug().Int("status", resp.StatusCode).Str("short_code", shortCode).Msg("Short URL not found")
		return nil, serviceErrors.NewNotFoundError("repository.GetByShortCode", "short URL not found")
	}

	var pbResp pocketBaseListResponse
	if err := json.NewDecoder(resp.Body).Decode(&pbResp); err != nil {
		log.Error().Err(err).Msg("Failed to decode response")
		return nil, serviceErrors.NewInternalError("repository.GetByShortCode", "failed to decode response", err)
	}

	if len(pbResp.Items) == 0 {
		log.Debug().Str("short_code", shortCode).Msg("Short URL not found")
		return nil, serviceErrors.NewNotFoundError("repository.GetByShortCode", "short URL not found")
	}

	record := pbResp.Items[0]
	shortURL := &urlModels.ShortURL{
		ID:          record.ID,
		URL:         record.URL,
		ShortCode:   record.ShortCode,
		AccessCount: record.AccessCount,
		Created:     parsePBTime(record.Created),
		Updated:     parsePBTime(record.Updated),
	}

	log.Debug().Str("short_code", shortCode).Str("url", record.URL).Msg("Short URL found")
	return shortURL, nil
}

func (r *urlRepositoryImpl) Update(ctx context.Context, shortCode string, newURL string) (*urlModels.ShortURL, error) {
	log.Debug().Str("short_code", shortCode).Str("new_url", newURL).Msg("Updating short URL")

	shortURL, err := r.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	if shortURL == nil {
		return nil, serviceErrors.NewNotFoundError("repository.Update", "record not found")
	}

	reqBody := pocketBaseUpdateRequest{
		URL: &newURL,
	}

	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
	defer cancel()

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, serviceErrors.NewInternalError("repository.Update", "failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		fmt.Sprintf("%s/api/collections/%s/records/%s", r.pb.BaseURL, constants.ShortURLsCollection, shortURL.ID),
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, serviceErrors.NewInternalError("repository.Update", "failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.pb.HTTPClient.Do(req)
	if err != nil {
		return nil, serviceErrors.NewInternalError("repository.Update", "failed to update record", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, serviceErrors.NewInternalError("repository.Update", "PocketBase error", fmt.Errorf("status %d", resp.StatusCode))
	}

	var pbResp pocketBaseRecord
	if err := json.NewDecoder(resp.Body).Decode(&pbResp); err != nil {
		return nil, serviceErrors.NewInternalError("repository.Update", "failed to decode response", err)
	}

	updatedURL := &urlModels.ShortURL{
		ID:          pbResp.ID,
		URL:         pbResp.URL,
		ShortCode:   pbResp.ShortCode,
		AccessCount: pbResp.AccessCount,
		Created:     parsePBTime(pbResp.Created),
		Updated:     parsePBTime(pbResp.Updated),
	}

	log.Info().Str("short_code", shortCode).Str("new_url", newURL).Msg("Short URL updated successfully")
	return updatedURL, nil
}

func (r *urlRepositoryImpl) Delete(ctx context.Context, shortCode string) error {
	log.Debug().Str("short_code", shortCode).Msg("Deleting short URL")

	shortURL, err := r.GetByShortCode(ctx, shortCode)
	if err != nil {
		return err
	}
	if shortURL == nil {
		return serviceErrors.NewNotFoundError("repository.Delete", "record not found")
	}

	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/api/collections/%s/records/%s", r.pb.BaseURL, constants.ShortURLsCollection, shortURL.ID),
		nil)
	if err != nil {
		return serviceErrors.NewInternalError("repository.Delete", "failed to create request", err)
	}

	resp, err := r.pb.HTTPClient.Do(req)
	if err != nil {
		return serviceErrors.NewInternalError("repository.Delete", "failed to delete record", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return serviceErrors.NewInternalError("repository.Delete", "PocketBase error", fmt.Errorf("status %d", resp.StatusCode))
	}

	log.Info().Str("short_code", shortCode).Msg("Short URL deleted successfully")
	return nil
}

func (r *urlRepositoryImpl) IncrementAccessCount(ctx context.Context, shortCode string) error {
	log.Debug().Str("short_code", shortCode).Msg("Incrementing access count")

	shortURL, err := r.GetByShortCode(ctx, shortCode)
	if err != nil {
		return err
	}
	if shortURL == nil {
		return serviceErrors.NewNotFoundError("repository.IncrementAccessCount", "record not found")
	}

	newCount := shortURL.AccessCount + 1
	reqBody := pocketBaseUpdateRequest{
		AccessCount: &newCount,
	}

	ctx, cancel := context.WithTimeout(ctx, constants.RequestTimeout)
	defer cancel()

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return serviceErrors.NewInternalError("repository.IncrementAccessCount", "failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		fmt.Sprintf("%s/api/collections/%s/records/%s", r.pb.BaseURL, constants.ShortURLsCollection, shortURL.ID),
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return serviceErrors.NewInternalError("repository.IncrementAccessCount", "failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.pb.HTTPClient.Do(req)
	if err != nil {
		return serviceErrors.NewInternalError("repository.IncrementAccessCount", "failed to update record", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return serviceErrors.NewInternalError("repository.IncrementAccessCount", "PocketBase error", fmt.Errorf("status %d", resp.StatusCode))
	}

	log.Info().Str("short_code", shortCode).Int64("new_count", newCount).Msg("Access count incremented")
	return nil
}

func (r *urlRepositoryImpl) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	log.Debug().Str("short_code", shortCode).Msg("Checking if short code exists")

	shortURL, err := r.GetByShortCode(ctx, shortCode)
	if err != nil {
		var serviceErr *serviceErrors.ServiceError
		if errors.As(err, &serviceErr) && serviceErr.Code == serviceErrors.ErrorCodeNotFound {
			return false, nil
		}
		return false, err
	}
	return shortURL != nil, nil
}
