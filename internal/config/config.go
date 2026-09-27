package config

import (
	"errors"
	"os"
)

type Config struct {
	Port             string
	DatabaseURL      string
	FirebaseCredFile string
}

func Load() (Config, error) {
	databaseURL, err := required("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	credentialsFile, err := required("FIREBASE_CREDENTIALS_FILE")
	if err != nil {
		return Config{}, err
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{Port: port, DatabaseURL: databaseURL, FirebaseCredFile: credentialsFile}, nil
}

func required(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", errors.New("la variable " + name + " es obligatoria")
	}
	return value, nil
}
