package web

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/config"
)

// AuthConfig is the config for the auth module.
type AuthConfig struct {
	// Provider contains a list of OAuth2 providers.
	Provider     map[string]ProviderConfig `toml:"provider"`
	EnableOAuth2 bool                      `toml:"enable_oauth2"`
}

// ProviderConfig is the config for an OAuth2 provider.
type ProviderConfig struct {
	Name           string `toml:"name"`
	AuthorizeURI   string `toml:"authorize_uri"`
	AccessTokenURI string `toml:"access_token_uri"`
	ClientID       string `toml:"client_id"`
	ClientSecret   string `toml:"client_secret"`
}

func LoadConfig(v *validator.Validate) {
	// TODO remove and implement real auth logic

	cfg := &AuthConfig{}
	err := config.C(cfg, config.From("auth"), config.Validate(v))
	if err != nil {
		fmt.Printf("failed to load auth config: %v", err)
	}

	fmt.Printf("config: %+v", cfg.EnableOAuth2)
}
