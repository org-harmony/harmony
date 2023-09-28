package web

import (
	"github.com/org-harmony/harmony"
)

// TODO write docs
type AuthCfg struct {
	Provider map[string]PCfg `toml:"provider"`
}

// TODO write docs
type PCfg struct {
	Name           string `toml:"name"`
	AuthorizeURI   string `toml:"authorize_uri"`
	AccessTokenURI string `toml:"access_token_uri"`
	ClientID       string `toml:"client_id"`
	ClientSecret   string `toml:"client_secret"`
}

func LoadConfig() {
	// TODO remove and implement real auth logic

	c := &AuthCfg{}
	err := harmony.Config("auth", c)
	if err != nil {
		return
	}
}
