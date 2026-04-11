package kleiogithub

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type testWorkspaceLookup struct{}

func (testWorkspaceLookup) FindByInstallationID(ctx context.Context, installationID int) (WorkspaceRef, error) {
	return WorkspaceRef{ID: "ws-test"}, nil
}

func (testWorkspaceLookup) LinkInstallation(ctx context.Context, installationID int, ownerLogin string) (WorkspaceRef, error) {
	return WorkspaceRef{ID: "ws-test"}, nil
}

type okRepo struct{}

func (okRepo) EnsureRepo(ctx context.Context, workspaceID string, repo RepoInfo) error { return nil }

func (okRepo) EnsureRepoShort(ctx context.Context, workspaceID string, repo RepoShortInfo) error { return nil }

type failRepo struct{}

func (failRepo) EnsureRepo(ctx context.Context, workspaceID string, repo RepoInfo) error {
	return errors.New("ensure repo failed")
}

func (failRepo) EnsureRepoShort(ctx context.Context, workspaceID string, repo RepoShortInfo) error { return nil }

type okCapture struct{}

func (okCapture) EmitGitCommit(ctx context.Context, workspaceID string, commit CommitPayload) error {
	return nil
}
func (okCapture) EmitGitPR(ctx context.Context, workspaceID string, pr PRPayload) error { return nil }
func (okCapture) EmitCIRun(ctx context.Context, workspaceID string, run CIRunPayload) error {
	return nil
}
func (okCapture) EmitDeployment(ctx context.Context, workspaceID string, deploy DeploymentPayload) error {
	return nil
}
func (okCapture) EmitDiscussion(ctx context.Context, workspaceID string, disc DiscussionPayload) error {
	return nil
}
func (okCapture) EmitSecurityAlert(ctx context.Context, workspaceID string, alert SecurityAlertPayload) error {
	return nil
}
func (okCapture) EmitPRReview(ctx context.Context, workspaceID string, review PRReviewPayload) error {
	return nil
}
func (okCapture) EmitPRReviewComment(ctx context.Context, workspaceID string, comment PRReviewCommentPayload) error {
	return nil
}
func (okCapture) EmitIssue(ctx context.Context, workspaceID string, issue IssuePayload) error {
	return nil
}

func hubSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	secret := "webhook-secret"
	body := []byte(`{"ref":"refs/heads/main"}`)
	sig := hubSig(secret, body)

	h := NewWebhookHandler(secret, nil, nil, nil)
	require.True(t, h.VerifySignature(body, sig))
	require.False(t, h.VerifySignature(body, "sha256=deadbeef"))

	hEmpty := NewWebhookHandler("", nil, nil, nil)
	require.True(t, hEmpty.VerifySignature(body, ""), "empty secret should skip verification (compat)")
}

func TestHandleHTTP_PushSuccess(t *testing.T) {
	secret := "s3cret"
	payload := map[string]any{
		"ref": "refs/heads/main",
		"repository": map[string]any{
			"id":             1,
			"full_name":      "o/r",
			"html_url":       "https://github.com/o/r",
			"default_branch": "main",
		},
		"commits": []any{
			map[string]any{
				"id":        "abcd1234ef567890abcd1234ef567890abcd1234",
				"message":   "msg",
				"author":    map[string]any{"name": "n", "email": "e@e"},
				"timestamp": "2020-01-01T00:00:00Z",
			},
		},
		"installation": map[string]any{"id": 42, "account": map[string]any{"login": "o"}},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	h := NewWebhookHandler(secret, testWorkspaceLookup{}, okRepo{}, okCapture{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "push")
	r.Header.Set("X-Hub-Signature-256", hubSig(secret, body))

	h.HandleHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
}

func TestHandleHTTP_InvalidPushJSONReturns400(t *testing.T) {
	secret := "s3cret"
	body := []byte(`{not json`)

	h := NewWebhookHandler(secret, testWorkspaceLookup{}, okRepo{}, okCapture{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "push")
	r.Header.Set("X-Hub-Signature-256", hubSig(secret, body))

	h.HandleHTTP(w, r)
	require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
}

func TestHandleHTTP_UnknownEventTypeReturnsOK(t *testing.T) {
	secret := "s3cret"
	body := []byte(`{}`)

	h := NewWebhookHandler(secret, testWorkspaceLookup{}, okRepo{}, okCapture{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "ping")
	r.Header.Set("X-Hub-Signature-256", hubSig(secret, body))

	h.HandleHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
}

func TestHandleHTTP_HandlerErrorReturns500(t *testing.T) {
	secret := "s3cret"
	payload := map[string]any{
		"ref": "refs/heads/main",
		"repository": map[string]any{
			"id":             1,
			"full_name":      "o/r",
			"html_url":       "https://github.com/o/r",
			"default_branch": "main",
		},
		"commits":        []any{},
		"installation": map[string]any{"id": 42, "account": map[string]any{"login": "o"}},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	h := NewWebhookHandler(secret, testWorkspaceLookup{}, failRepo{}, okCapture{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(body))
	r.Header.Set("X-GitHub-Event", "push")
	r.Header.Set("X-Hub-Signature-256", hubSig(secret, body))

	h.HandleHTTP(w, r)
	require.Equal(t, http.StatusInternalServerError, w.Code, "body: %s", w.Body.String())
}
