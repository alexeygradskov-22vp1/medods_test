package external

import (
	"fmt"
	"go.dataddo.com/env"
	"log"
	"net/http"
)

type ExternalClientConfig struct {
	BaseUrl string `env:"BASE_URL"`
	client  *http.Client
}

func NewConfig(client *http.Client) (*ExternalClientConfig, error) {
	var config ExternalClientConfig
	if err := env.Load(&config, "EXTERNAL_CLIENT_"); err != nil {
		return nil, fmt.Errorf("an error occurred while loading the database config: %v", err)
	}
	log.Printf("external config was loaded successfully")
	return &config, nil
}
