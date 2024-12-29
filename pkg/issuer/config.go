package issuer

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"
)

type ConfigLoader func() (Config, error)

type Config struct {
	Endpoint  string
	Values    url.Values
	DstPath   string
	BasicAuth string

	RefreshBefore time.Duration
}

const envPrefix = "dex_"
const envFromFilePrefix = "dex_file_"

var _ ConfigLoader = ConfigFromEnvs

func ConfigFromEnvs() (Config, error) {
	config := Config{
		Values: url.Values{},
	}
	for _, env := range os.Environ() {
		key, value, err := loadFromFile(env)
		if err != nil {
			return Config{}, err
		}

		if key != "" {
			config.Values.Add(key, value)
			continue
		}

		key, value, err = loadFromEnv(env)
		if err != nil {
			return Config{}, err
		}

		if key != "" {
			config.Values.Add(key, value)
			continue
		}
	}

	config.DstPath = os.Getenv("dex_access_token_file")
	config.Endpoint = os.Getenv("dex_endpoint")
	config.BasicAuth = os.Getenv("dex_basic_auth")

	var err error
	if d := os.Getenv("dex_refresh_before"); d != "" {
		config.RefreshBefore, err = time.ParseDuration(d)
		if err != nil {
			return Config{}, fmt.Errorf("failed to parse dex_refresh_before: %w", err)
		}
	} else {
		config.RefreshBefore = 1 * time.Hour
	}

	slog.Info("config loaded", "config", config)

	return config, nil
}

func loadFromEnv(env string) (key, value string, err error) {
	after, found := strings.CutPrefix(env, envPrefix)
	if !found {
		return "", "", nil
	}

	key, value, _ = strings.Cut(after, "=")

	return key, value, nil
}

func loadFromFile(env string) (key, value string, err error) {
	after, found := strings.CutPrefix(env, envFromFilePrefix)
	if !found {
		return "", "", nil
	}

	key, value, _ = strings.Cut(after, "=")

	b, err := os.ReadFile(value)
	if err != nil {
		return "", "", fmt.Errorf("failed to open file %s: %w", after, err)
	}

	return key, string(b), nil
}

/*
TOKEN=$(curl -v https://dex.tsuzu.dev/token
	 --user incus:
	 --data-urlencode "connector_id=k3s"
	 --data-urlencode "grant_type=urn:ietf:params:oauth:grant-type:token-exchange"
	 --data-urlencode "scope=openid federated:id"
	 --data-urlencode "requested_token_type=urn:ietf:params:oauth:token-type:access_token"
	 --data-urlencode "subject_token=$UPSTREAM_TOKEN"
	 --data-urlencode "subject_token_type=urn:ietf:params:oauth:token-type:id_token" | jq -r .access_token)
*/
