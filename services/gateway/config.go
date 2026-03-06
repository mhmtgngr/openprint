package main

import (
	"fmt"
	"log"
	"os"
)

// Config holds gateway configuration.
type Config struct {
	ServerAddr             string
	JWTSecret              string
	RequestsPerMinute      int
	ServiceHost            string
	AuthServiceURL         string
	RegistryServiceURL     string
	JobServiceURL          string
	StorageServiceURL      string
	NotificationServiceURL string
}

func loadConfig() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long")
	}

	serviceHost := getEnv("SERVICE_HOST", "localhost")
	requestsPerMinute := getEnvInt("REQUESTS_PER_MINUTE", 100)

	return &Config{
		ServerAddr:             fmt.Sprintf(":%d", ServicePort),
		JWTSecret:              jwtSecret,
		RequestsPerMinute:      requestsPerMinute,
		ServiceHost:            serviceHost,
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, AuthServicePort)),
		RegistryServiceURL:     getEnv("REGISTRY_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, RegistryServicePort)),
		JobServiceURL:          getEnv("JOB_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, JobServicePort)),
		StorageServiceURL:      getEnv("STORAGE_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, StorageServicePort)),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, NotificationServicePort)),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}
