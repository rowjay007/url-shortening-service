package validator

import (
	"github.com/rowjay/url-shortening-service/internal/constants"
	"github.com/rowjay/url-shortening-service/internal/errors"
	"net/url"
	"slices"
	"strings"
)

type URLValidator struct {
	maxURLLength   int
	blockedDomains []string
}

func NewURLValidator() *URLValidator {
	return &URLValidator{
		maxURLLength:   constants.MaxURLLength,
		blockedDomains: constants.BlockedDomains,
	}
}

func (v *URLValidator) ValidateURL(rawURL string) error {
	if len(rawURL) > v.maxURLLength {
		return errors.NewValidationError("validator.ValidateURL", "URL too long", nil)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.NewValidationError("validator.ValidateURL", "invalid URL format", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.NewValidationError("validator.ValidateURL", "only HTTP and HTTPS URLs are allowed", nil)
	}

	if v.isDomainBlocked(parsed.Host) {
		return errors.NewValidationError("validator.ValidateURL", "domain is blocked", nil)
	}

	return nil
}

func (v *URLValidator) ValidateShortCode(code string) error {
	if len(code) < 4 || len(code) > 20 {
		return errors.NewValidationError("validator.ValidateShortCode", "short code must be between 4 and 20 characters", nil)
	}

	for _, char := range code {
		if !isAlphaNumeric(char) {
			return errors.NewValidationError("validator.ValidateShortCode", "short code can only contain alphanumeric characters", nil)
		}
	}

	return nil
}

func (v *URLValidator) isDomainBlocked(domain string) bool {
	domain = strings.ToLower(domain)
	return slices.Contains(v.blockedDomains, domain)
}

func isAlphaNumeric(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}
