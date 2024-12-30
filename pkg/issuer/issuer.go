package issuer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Issuer interface {
	Rotate(ctx context.Context) error
}

func NewIssuer(loader ConfigLoader) Issuer {
	return &tokenIssuer{
		client: http.Client{
			Timeout: 10 * time.Second,
		},
		loader: loader,
		now:    time.Now,
	}
}

type tokenIssuer struct {
	client http.Client
	loader ConfigLoader
	now    func() time.Time
}

type tokenResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
}

func (iss *tokenIssuer) issue(ctx context.Context, config Config) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", config.Endpoint, strings.NewReader(config.Values.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}
	if config.BasicAuth != "" {
		user, password, _ := strings.Cut(config.BasicAuth, ":")
		req.SetBasicAuth(user, password)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := iss.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send a request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)

		slog.Error("failed to issue access token", "body", string(b))

		return "", 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return token.AccessToken, 0, nil
}

func (iss *tokenIssuer) loadTokenExp(ctx context.Context, config Config) (*time.Time, error) {
	b, err := os.ReadFile(config.DstPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", config.DstPath, err)
	}

	var claims jwt.RegisteredClaims
	_, err = jwt.ParseWithClaims(string(b), &claims, nil, jwt.WithoutClaimsValidation())
	if err != nil {
		return nil, fmt.Errorf("failed to parse jwt: %w", err)
	}

	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("missing expiration claim")
	}

	return &claims.ExpiresAt.Time, nil
}

func (iss *tokenIssuer) save(ctx context.Context, config Config, token string) error {
	fp, err := os.Create(config.DstPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", config.DstPath, err)
	}
	defer fp.Close()

	_, err = fp.WriteString(token)
	if err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}

	return nil
}

func (iss *tokenIssuer) Issue(ctx context.Context) (string, error) {
	config, err := iss.loader()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	token, _, err := iss.issue(ctx, config)
	if err != nil {
		return "", fmt.Errorf("failed to issue token: %w", err)
	}

	return token, nil
}

func (iss *tokenIssuer) Rotate(ctx context.Context) error {
	config, err := iss.loader()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	expiresAt, err := iss.loadTokenExp(ctx, config)
	if err != nil {
		slog.Warn("failed to load token expiration", "error", err)
	} else if expiresAt.After(iss.now().Add(config.RefreshBefore)) {
		return nil
	}
	slog.Info("token is being outdated or deleted", "expires_at", expiresAt, "refresh_before", config.RefreshBefore)

	token, expIn, err := iss.issue(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to issue token: %w", err)
	}

	exp := iss.now().Add(time.Duration(expIn) * time.Second)
	refreshedAt := iss.now().Add(config.RefreshBefore)
	if exp.Before(refreshedAt) {
		slog.Warn("token expiration is too short", "expires_at", exp, "refresh_scheduled_at", refreshedAt, "refresh_before", config.RefreshBefore)
	}

	if err := iss.save(ctx, config, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	slog.Info("token refreshed", "expires_at", exp, "path", config.DstPath)

	return nil
}
