package kleiogithub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"
)

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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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

func fetchRepoPage(ctx context.Context, accessToken, url string) ([]Repo, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, true, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
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
