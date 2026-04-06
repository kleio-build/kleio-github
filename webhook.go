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
	case "workflow_run":
		var event WorkflowRunEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleWorkflowRun(ctx, &event); err != nil {
			fmt.Printf("webhook workflow_run error: %v\n", err)
		}
	case "deployment_status":
		var event DeploymentStatusEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleDeploymentStatus(ctx, &event); err != nil {
			fmt.Printf("webhook deployment_status error: %v\n", err)
		}
	case "discussion":
		var event DiscussionEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleDiscussion(ctx, &event); err != nil {
			fmt.Printf("webhook discussion error: %v\n", err)
		}
	case "dependabot_alert":
		var event DependabotAlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleDependabotAlert(ctx, &event); err != nil {
			fmt.Printf("webhook dependabot_alert error: %v\n", err)
		}
	case "code_scanning_alert":
		var event CodeScanningAlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleCodeScanningAlert(ctx, &event); err != nil {
			fmt.Printf("webhook code_scanning_alert error: %v\n", err)
		}
	case "pull_request_review":
		var event PullRequestReviewEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandlePullRequestReview(ctx, &event); err != nil {
			fmt.Printf("webhook pull_request_review error: %v\n", err)
		}
	case "security_advisory":
		var event SecurityAdvisoryEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleSecurityAdvisory(ctx, &event); err != nil {
			fmt.Printf("webhook security_advisory error: %v\n", err)
		}
	case "issues":
		var event IssuesEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleIssues(ctx, &event); err != nil {
			fmt.Printf("webhook issues error: %v\n", err)
		}
	case "check_suite":
		var event CheckSuiteEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleCheckSuite(ctx, &event); err != nil {
			fmt.Printf("webhook check_suite error: %v\n", err)
		}
	case "check_run":
		var event CheckRunEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if err := h.HandleCheckRun(ctx, &event); err != nil {
			fmt.Printf("webhook check_run error: %v\n", err)
		}
	case "status":
		fmt.Printf("status event: state=%s\n", "received")
	case "installation_target", "meta":
		fmt.Printf("infrastructure event: %s\n", eventType)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
