// Package config reads the shop's settings from the environment.
package config

import (
	"os"

	"example.com/bazelviewer/pkg/validate"
)

// Config holds the shop's settings.
type Config struct {
	Addr        string // SHOP_ADDR: HTTP listen address.
	DatabaseURL string // SHOP_DB: database to open.
	MailFrom    string // SHOP_MAIL_FROM: sender of customer emails.
}

// Load reads the settings, with defaults for local runs.
func Load() (Config, error) {
	c := Config{
		Addr:        env("SHOP_ADDR", ":8080"),
		DatabaseURL: env("SHOP_DB", "memory://"),
		MailFrom:    env("SHOP_MAIL_FROM", "shop@example.com"),
	}
	if err := validate.Email(c.MailFrom); err != nil {
		return c, err
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
