package services

import (
	"context"

	"github.com/rowjay/url-shortening-service/internal/dto"
	"github.com/rowjay/url-shortening-service/internal/errors"
	"github.com/rowjay/url-shortening-service/internal/models"
	"github.com/rowjay/url-shortening-service/internal/repository"
	"github.com/rowjay/url-shortening-service/internal/utils"
	"github.com/rowjay/url-shortening-service/internal/validator"
)

type URLService interface {
	CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string) (*dto.GetURLResponse, error)
	UpdateShortURL(ctx context.Context, shortCode string, req *dto.UpdateURLRequest) (*dto.UpdateURLResponse, error)
	DeleteShortURL(ctx context.Context, shortCode string) error
	GetStatistics(ctx context.Context, shortCode string) (*dto.GetStatsResponse, error)
}

type urlServiceImpl struct {
	repo      repository.URLRepository
	validator *validator.URLValidator
}

func NewURLService(repo repository.URLRepository) URLService {
	return &urlServiceImpl{
		repo:      repo,
		validator: validator.NewURLValidator(),
	}
}

func (s *urlServiceImpl) CreateShortURL(ctx context.Context, req *dto.CreateURLRequest) (*dto.CreateURLResponse, error) {
	if err := s.validator.ValidateURL(req.URL); err != nil {
		return nil, err
	}

	var shortCode string
	if req.CustomCode != nil {
		if err := s.validator.ValidateShortCode(*req.CustomCode); err != nil {
			return nil, err
		}
		
		exists, err := s.repo.ExistsByShortCode(ctx, *req.CustomCode)
		if err != nil {
			return nil, errors.NewInternalError("service.CreateShortURL", "failed to check code existence", err)
		}
		if exists {
			return nil, errors.NewDuplicateError("service.CreateShortURL", "short code already exists")
		}
		shortCode = *req.CustomCode
	} else {
		var err error
		shortCode, err = s.generateUniqueShortCode(ctx)
		if err != nil {
			return nil, errors.NewInternalError("service.CreateShortURL", "failed to generate short code", err)
		}
	}

	shortURL := &models.ShortURL{
		URL:         req.URL,
		ShortCode:   shortCode,
		AccessCount: 0,
	}

	if err := s.repo.Create(ctx, shortURL); err != nil {
		return nil, err
	}

	return &dto.CreateURLResponse{
		ID:        shortURL.ID,
		URL:       shortURL.URL,
		ShortCode: shortURL.ShortCode,
		CreatedAt: shortURL.Created,
		UpdatedAt: shortURL.Updated,
	}, nil
}

func (s *urlServiceImpl) GetOriginalURL(ctx context.Context, shortCode string) (*dto.GetURLResponse, error) {
	shortURL, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	if err := s.repo.IncrementAccessCount(ctx, shortCode); err != nil {
		return nil, errors.NewInternalError("service.GetOriginalURL", "failed to increment access count", err)
	}

	return &dto.GetURLResponse{
		ID:        shortURL.ID,
		URL:       shortURL.URL,
		ShortCode: shortURL.ShortCode,
		CreatedAt: shortURL.Created,
		UpdatedAt: shortURL.Updated,
	}, nil
}

func (s *urlServiceImpl) UpdateShortURL(ctx context.Context, shortCode string, req *dto.UpdateURLRequest) (*dto.UpdateURLResponse, error) {
	if err := s.validator.ValidateURL(req.URL); err != nil {
		return nil, err
	}

	updatedURL, err := s.repo.Update(ctx, shortCode, req.URL)
	if err != nil {
		return nil, err
	}

	return &dto.UpdateURLResponse{
		ID:          updatedURL.ID,
		URL:         updatedURL.URL,
		ShortCode:   updatedURL.ShortCode,
		AccessCount: updatedURL.AccessCount,
		CreatedAt:   updatedURL.Created,
		UpdatedAt:   updatedURL.Updated,
	}, nil
}

func (s *urlServiceImpl) DeleteShortURL(ctx context.Context, shortCode string) error {
	return s.repo.Delete(ctx, shortCode)
}

func (s *urlServiceImpl) GetStatistics(ctx context.Context, shortCode string) (*dto.GetStatsResponse, error) {
	shortURL, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	return &dto.GetStatsResponse{
		ID:          shortURL.ID,
		URL:         shortURL.URL,
		ShortCode:   shortURL.ShortCode,
		AccessCount: shortURL.AccessCount,
		CreatedAt:   shortURL.Created,
		UpdatedAt:   shortURL.Updated,
	}, nil
}

func (s *urlServiceImpl) generateUniqueShortCode(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := utils.GenerateShortCode(6)
		if err != nil {
			return "", errors.NewInternalError("service.generateUniqueShortCode", "failed to generate short code", err)
		}
		exists, err := s.repo.ExistsByShortCode(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.NewInternalError("service.generateUniqueShortCode", "failed to generate unique code after 10 attempts", nil)
}
