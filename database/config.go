package database

import (
	"fmt"
	"go.dataddo.com/env"
	"log"
)

type DBConfig struct {
	Host     string `env:"HOST"`
	Port     string `env:"PORT"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	DB       string `env:"DB"`
}

func NewConfig() (*DBConfig, error) {
	var config DBConfig
	if err := env.Load(&config, "PG_"); err != nil {
		return nil, fmt.Errorf("an error occurred while loading the database config: %v", err)
	}
	log.Printf("db config was loaded successfully")
	return &config, nil
}
