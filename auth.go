package kleiogithub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"
)

// githubAPIHTTPClient bounds outbound GitHub REST calls (avoids hanging forever).
var githubAPIHTTPClient = &http.Client{Timeout: 60 * time.Second}

// AuthConfig configures the GitHub OAuth / GitHub App user-authorization flow.
type AuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	// Scopes are OAuth scopes for OAuth Apps. For GitHub Apps, leave empty —
	// permissions are configured on the App itself.
	Scopes []string
}

// AuthClient handles GitHub user authentication via OAuth2 / GitHub App
// user-authorization. It does NOT handle database operations — callers
// receive raw GitHub data and manage persistence themselves.
type AuthClient struct {
	oauth *oauth2.Config
}

// NewAuthClient creates an AuthClient from the given config.
func NewAuthClient(cfg AuthConfig) *AuthClient {
	return &AuthClient{
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Scopes:       cfg.Scopes,
			Endpoint:     oauthgithub.Endpoint,
			RedirectURL:  cfg.RedirectURL,
		},
	}
}

// GetAuthURL returns the GitHub authorization URL with the given state parameter.
func (a *AuthClient) GetAuthURL(state string) string {
	return a.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// AuthResult is returned after a successful code exchange.
type AuthResult struct {
	User        *User
	AccessToken string
	// RefreshToken is populated for GitHub App user-to-server tokens (8hr TTL).
	// Empty for legacy OAuth App tokens.
	RefreshToken string
}

// ExchangeCode exchanges an authorization code for tokens and fetches the
// authenticated GitHub user profile.
func (a *AuthClient) ExchangeCode(ctx context.Context, code string) (*AuthResult, error) {
	token, err := a.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	user, err := FetchUser(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("fetch github user: %w", err)
	}

	return &AuthResult{
		User:         user,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}, nil
}

// RefreshUserToken exchanges a GitHub App refresh token for a new access token.
// This is only applicable for GitHub App user-to-server tokens (not legacy OAuth).
func (a *AuthClient) RefreshUserToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	tokenSource := a.oauth.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh github token: %w", err)
	}
	return newToken, nil
}

// FetchUser retrieves the authenticated user's GitHub profile.
func FetchUser(ctx context.Context, accessToken string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// FetchUserOrgs lists the authenticated user's GitHub organizations.
func FetchUserOrgs(ctx context.Context, accessToken string) ([]Org, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/orgs?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github orgs API returned %d: %s", resp.StatusCode, string(body))
	}

	var orgs []Org
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

// FetchOrgMembership returns the user's role in a given org ("admin" or "member").
func FetchOrgMembership(ctx context.Context, accessToken, org, username string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/memberships/%s", org, username)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "member", nil
	}

	var membership struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&membership); err != nil {
		return "member", nil
	}
	return membership.Role, nil
}

// FetchOrgRepos lists all repositories for a GitHub organization, paginated.
func FetchOrgRepos(ctx context.Context, accessToken, org string) ([]Repo, error) {
	var all []Repo
	page := 1
	for {
		repos, done, err := fetchRepoPage(ctx, accessToken,
			fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&page=%d&type=all", org, page))
		if err != nil {
			return all, err
		}
		all = append(all, repos...)
		if done {
			break
		}
		page++
	}
	return all, nil
}

// FetchUserRepos lists repositories owned by the authenticated user, paginated.
func FetchUserRepos(ctx context.Context, accessToken string) ([]Repo, error) {
	var all []Repo
	page := 1
	for {
		repos, done, err := fetchRepoPage(ctx, accessToken,
			fmt.Sprintf("https://api.github.com/user/repos?per_page=100&page=%d&affiliation=owner&sort=updated", page))
		if err != nil {
			return all, err
		}
		all = append(all, repos...)
		if done {
			break
		}
		page++
	}
	return all, nil
}

// AppInstallation represents a GitHub App installation accessible to the user.
type AppInstallation struct {
	ID      int              `json:"id"`
	Account AppInstallTarget `json:"account"`
	AppSlug string           `json:"app_slug"`
}

// AppInstallTarget is the account (org or user) that owns an installation.
type AppInstallTarget struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// FetchUserInstallations lists GitHub App installations accessible to the
// authenticated user. Returns a map of account login -> installation ID for
// easy lookup during onboarding.
func FetchUserInstallations(ctx context.Context, accessToken string) (map[string]int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.github.com/user/installations?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var result struct {
		Installations []AppInstallation `json:"installations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	instMap := make(map[string]int, len(result.Installations))
	for _, inst := range result.Installations {
		instMap[inst.Account.Login] = inst.ID
	}
	return instMap, nil
}

// GitHubEmail represents an email address from the /user/emails endpoint.
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// FetchUserEmails retrieves the authenticated user's email addresses.
// Returns the primary verified email, falling back to any verified email.
func FetchUserEmails(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails API returned %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	var fallback string
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
		if e.Verified && fallback == "" {
			fallback = e.Email
		}
	}
	return fallback, nil
}

// DeviceCodeResponse is returned by GitHub's device authorization endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceTokenResponse is returned when polling GitHub's token endpoint during
// the device flow. On success, AccessToken is populated. While the user hasn't
// authorized yet, Error is "authorization_pending" or "slow_down".
type DeviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
	Interval     int    `json:"interval,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// RequestDeviceCode initiates a GitHub device authorization flow. The returned
// DeviceCodeResponse contains the user_code to display and verification_uri to
// open in a browser. Only requires client_id (no secret).
func RequestDeviceCode(ctx context.Context, clientID string) (*DeviceCodeResponse, error) {
	form := url.Values{
		"client_id": {clientID},
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/device/code", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = form.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request returned %d: %s", resp.StatusCode, string(body))
	}

	var result DeviceCodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse device code response: %w", err)
	}
	if result.DeviceCode == "" {
		return nil, fmt.Errorf("empty device_code in response: %s", string(body))
	}
	return &result, nil
}

// PollDeviceToken polls GitHub's token endpoint for a device flow authorization.
// Returns a DeviceTokenResponse whose Error field indicates the current state:
//   - "": success, AccessToken is populated
//   - "authorization_pending": user hasn't authorized yet
//   - "slow_down": caller should increase polling interval
//   - "expired_token": device code expired
//   - "access_denied": user cancelled
func PollDeviceToken(ctx context.Context, clientID, deviceCode string) (*DeviceTokenResponse, error) {
	form := url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = form.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device token poll: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device token poll returned %d: %s", resp.StatusCode, string(body))
	}

	var result DeviceTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse device token response: %w", err)
	}
	return &result, nil
}

func fetchRepoPage(ctx context.Context, accessToken, url string) ([]Repo, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, true, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := githubAPIHTTPClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, true, fmt.Errorf("github repos API returned %d: %s", resp.StatusCode, string(body))
	}

	var repos []Repo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, true, err
	}
	return repos, len(repos) < 100, nil
}
