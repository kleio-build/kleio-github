package kleiogithub

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WebhookHandler processes inbound GitHub webhook events. It delegates
// domain operations (capture creation, repo management) to injected interfaces.
type WebhookHandler struct {
	secret    string
	workspace WorkspaceLookup
	repos     RepoStore
	captures  CaptureEmitter
}

// NewWebhookHandler creates a WebhookHandler with the given dependencies.
func NewWebhookHandler(secret string, workspace WorkspaceLookup, repos RepoStore, captures CaptureEmitter) *WebhookHandler {
	return &WebhookHandler{
		secret:    secret,
		workspace: workspace,
		repos:     repos,
		captures:  captures,
	}
}

// VerifySignature checks the X-Hub-Signature-256 header against the payload.
// Returns true if the secret is empty (unconfigured) for backward compatibility.
func (h *WebhookHandler) VerifySignature(payload []byte, signature string) bool {
	if h.secret == "" {
		return true
	}
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// HandleHTTP is a convenience method that reads the request body, verifies
// the signature, dispatches to the correct handler, and writes a response.
// Use this to wire into any HTTP router.
func (h *WebhookHandler) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Hub-Signature-256")
	if !h.VerifySignature(body, sig) {
		http.Error(w, `{"error":"invalid signature"}`, http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	ctx := r.Context()

	switch eventType {
	case "push":
		var event PushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandlePush(ctx, &event); err != nil {
			fmt.Printf("webhook push error: %v\n", err)
		}
	case "pull_request":
		var event PullRequestEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandlePullRequest(ctx, &event); err != nil {
			fmt.Printf("webhook PR error: %v\n", err)
		}
	case "installation":
		var event InstallationEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleInstallation(ctx, &event); err != nil {
			fmt.Printf("webhook installation error: %v\n", err)
		}
	case "installation_repositories":
		var event InstallationReposEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleInstallationRepos(ctx, &event); err != nil {
			fmt.Printf("webhook installation repos error: %v\n", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
