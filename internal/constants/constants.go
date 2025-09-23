package constants

import "time"

const (
	ShortURLsCollection    = "short_urls"
	DefaultPageSize        = 30
	DefaultShortCodeLength = 6
	MaxRetries             = 5
	RequestTimeout         = 30 * time.Second
	MaxURLLength           = 2048
)

var BlockedDomains = []string{
	"malware.com",
	"phishing.com",
}
