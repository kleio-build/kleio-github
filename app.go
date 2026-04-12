package kleiogithub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AppConfig configures the GitHub App service.
type AppConfig struct {
	AppID     int64
	// SignerURL is the URL of the token-signing service that holds the private key.
	// The main application never has the PEM — it requests scoped installation
	// tokens from the signer.
	SignerURL string
}

// AppService manages GitHub App installation tokens via a dedicated signer service.
// It caches tokens with a 50-minute TTL (tokens are valid for 60 minutes).
type AppService struct {
	config     AppConfig
	httpClient *http.Client
	cache      sync.Map
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

// NewAppService creates a new AppService.
func NewAppService(cfg AppConfig) *AppService {
	cfg.SignerURL = strings.TrimSuffix(strings.TrimSpace(cfg.SignerURL), "/")
	return &AppService{
		config:     cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// MintTokenRequest is sent to the signer service.
type MintTokenRequest struct {
	InstallationID int64             `json:"installation_id"`
	Repositories   []string          `json:"repositories,omitempty"`
	Permissions    map[string]string `json:"permissions,omitempty"`
}

// MintTokenResponse is returned by the signer service.
type MintTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetInstallationToken returns a scoped installation token. Tokens are cached
// for 50 minutes. Pass repositories and permissions to scope the token —
// even if a token leaks, it can only access the specified repos/perms.
func (s *AppService) GetInstallationToken(ctx context.Context, installationID int64, repos []string, perms map[string]string) (string, error) {
	cacheKey := fmt.Sprintf("%d", installationID)

	if cached, ok := s.cache.Load(cacheKey); ok {
		ct := cached.(cachedToken)
		if time.Now().Before(ct.expiresAt) {
			return ct.token, nil
		}
		s.cache.Delete(cacheKey)
	}

	body, _ := json.Marshal(MintTokenRequest{
		InstallationID: installationID,
		Repositories:   repos,
		Permissions:    perms,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.SignerURL+"/mint-token", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create signer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call signer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("signer returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result MintTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode signer response: %w", err)
	}

	s.cache.Store(cacheKey, cachedToken{
		token:     result.Token,
		expiresAt: time.Now().Add(50 * time.Minute),
	})

	return result.Token, nil
}
