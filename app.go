package kleiogithub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	idToken, idErr := fetchGCPIdentityToken(ctx, s.config.SignerURL)
	switch {
	case s.config.SignerURL != "" && !isLocalSignerURL(s.config.SignerURL):
		// Cloud Run → Cloud Run requires a Google ID token; the audience query param
		// must be URL-encoded or the metadata server rejects the request and we
		// would call the signer with no Authorization header (403).
		if idErr != nil {
			return "", fmt.Errorf("get id token for signer: %w", idErr)
		}
		if idToken == "" {
			return "", fmt.Errorf("get id token for signer: empty token")
		}
		req.Header.Set("Authorization", "Bearer "+idToken)
	case idToken != "":
		req.Header.Set("Authorization", "Bearer "+idToken)
	}

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

func isLocalSignerURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// fetchGCPIdentityToken retrieves an OIDC identity token from the GCE metadata
// server. On Cloud Run / GCE / GKE the server is always available; on local dev
// it is not, so callers should treat errors as non-fatal.
//
// The audience must be passed as a properly encoded query value; a raw URL
// (e.g. https://....run.app) must not be concatenated unencoded, or the
// metadata server may see a truncated audience and return an error.
func fetchGCPIdentityToken(ctx context.Context, audience string) (string, error) {
	if strings.TrimSpace(audience) == "" {
		return "", nil
	}
	base, err := url.Parse("http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity")
	if err != nil {
		return "", err
	}
	q := base.Query()
	q.Set("audience", audience)
	base.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", base.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return strings.TrimSpace(string(body)), nil
}
