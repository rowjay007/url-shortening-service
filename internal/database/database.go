package database

import (
	"github.com/rs/zerolog/log"
	"net/http"
)

type PBClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func Initialize(pocketbaseURL string) (*PBClient, error) {
	log.Info().Str("url", pocketbaseURL).Msg("Initializing PocketBase HTTP client")

	client := &PBClient{
		BaseURL:    pocketbaseURL,
		HTTPClient: &http.Client{},
	}

	// Test connection to PocketBase
	resp, err := client.HTTPClient.Get(pocketbaseURL + "/api/health")
	if err != nil {
		log.Warn().Err(err).Msg("Could not connect to PocketBase - make sure it's running")
		// Don't fail here, just warn
	} else {
		resp.Body.Close()
		log.Info().Msg("Successfully connected to PocketBase")
	}

	return client, nil
}

func (pb *PBClient) StartServer() error {
	// PocketBase runs as a separate service now
	log.Info().Msg("PocketBase should be started separately with: ./pocketbase serve")
	return nil
}

func (pb *PBClient) CreateCollection() error {
	log.Info().Msg("Collection should be created through PocketBase admin UI at http://localhost:8090/_/")
	log.Info().Msg("Create a 'short_urls' collection with fields: url (text, required), short_code (text, required, unique), access_count (number, default: 0)")
	return nil
}
