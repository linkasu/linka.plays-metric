package app

import (
	"errors"
	"os"

	"github.com/linkasu/linka.plays-metric/internal/auth"
)

func HMACKeyring(prefix string, fallbackSecret []byte) (auth.ServiceKey, *auth.ServiceKey, error) {
	activeID := os.Getenv(prefix + "_ACTIVE_KEY_ID")
	activeSecret := os.Getenv(prefix + "_ACTIVE_SECRET")
	if activeID == "" {
		activeID = "default"
	}
	if activeSecret == "" {
		activeSecret = string(fallbackSecret)
	}
	active := auth.ServiceKey{ID: activeID, Secret: []byte(activeSecret)}

	previousID := os.Getenv(prefix + "_PREVIOUS_KEY_ID")
	previousSecret := os.Getenv(prefix + "_PREVIOUS_SECRET")
	if previousID == "" && previousSecret == "" {
		return active, nil, nil
	}
	if previousID == "" || previousSecret == "" {
		return auth.ServiceKey{}, nil, errors.New(prefix + " previous key ID and secret must be configured together")
	}
	previous := auth.ServiceKey{ID: previousID, Secret: []byte(previousSecret)}
	return active, &previous, nil
}
