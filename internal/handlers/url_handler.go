package handlers

import (
	"errors" 
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/rowjay/url-shortening-service/internal/dto"
	serviceErrors "github.com/rowjay/url-shortening-service/internal/errors"
	"github.com/rowjay/url-shortening-service/internal/services"
)

type URLHandler struct {
	service services.URLService
}

func NewURLHandler(service services.URLService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) CreateShortURL(c *gin.Context) {
	var req dto.CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid request payload")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request payload",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	resp, err := h.service.CreateShortURL(c.Request.Context(), &req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	log.Info().Str("short_code", resp.ShortCode).Str("url", resp.URL).Msg("Short URL created successfully")
	c.JSON(http.StatusCreated, resp)
}

func (h *URLHandler) GetOriginalURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Missing short code",
			Message: "Short code parameter is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	resp, err := h.service.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *URLHandler) UpdateShortURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Missing short code",
			Message: "Short code parameter is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req dto.UpdateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("Invalid request payload")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request payload",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	resp, err := h.service.UpdateShortURL(c.Request.Context(), shortCode, &req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	log.Info().Str("short_code", shortCode).Str("new_url", req.URL).Msg("Short URL updated successfully")
	c.JSON(http.StatusOK, resp)
}

func (h *URLHandler) DeleteShortURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Missing short code",
			Message: "Short code parameter is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.service.DeleteShortURL(c.Request.Context(), shortCode)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	log.Info().Str("short_code", shortCode).Msg("Short URL deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Short URL deleted successfully"})
}

func (h *URLHandler) GetStatistics(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Missing short code",
			Message: "Short code parameter is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	resp, err := h.service.GetStatistics(c.Request.Context(), shortCode)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *URLHandler) handleServiceError(c *gin.Context, err error) {
	var serviceErr *serviceErrors.ServiceError
	if errors.As(err, &serviceErr) {
		var statusCode int
		switch serviceErr.Code {
		case serviceErrors.ErrorCodeNotFound:
			statusCode = http.StatusNotFound
		case serviceErrors.ErrorCodeDuplicate:
			statusCode = http.StatusConflict
		case serviceErrors.ErrorCodeValidation:
			statusCode = http.StatusBadRequest
		case serviceErrors.ErrorCodeBadRequest:
			statusCode = http.StatusBadRequest
		default:
			statusCode = http.StatusInternalServerError
		}

		log.Error().Err(err).Int("status", statusCode).Msg("Service error")
		c.JSON(statusCode, dto.ErrorResponse{
			Error:   serviceErr.Message,
			Message: serviceErr.Error(),
			Code:    statusCode,
		})
		return
	}

	log.Error().Err(err).Msg("Unknown error")
	c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
		Error:   "Internal server error",
		Message: "An unexpected error occurred",
		Code:    http.StatusInternalServerError,
	})
}
