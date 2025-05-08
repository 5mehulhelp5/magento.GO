package config

import (
	"os"
	"sync"
)

// AppConfig holds global application configuration
var AppConfig *Config
var once sync.Once

type Config struct {
	AppName   string
	Port      string
	Env       string
	Debug     bool
	MediaUrl  string
	// Add more fields as needed
}

// LoadAppConfig initializes the global AppConfig variable
func LoadAppConfig() {
	once.Do(func() {
		AppConfig = &Config{
			AppName: os.Getenv("APP_NAME"),
			Port:    os.Getenv("PORT"),
			Env:     os.Getenv("APP_ENV"),
			Debug:   os.Getenv("DEBUG") == "true",
			MediaUrl: "https://react-luma.cnxt.link/media/catalog/product/",
		}
	})
} 