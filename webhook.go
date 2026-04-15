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
// Returns false if the secret is empty — webhooks must not be accepted without HMAC verification.
func (h *WebhookHandler) VerifySignature(payload []byte, signature string) bool {
	if h.secret == "" {
		return false
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

	var dispatchErr error
	switch eventType {
	case "push":
		var event PushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandlePush(ctx, &event)
	case "pull_request":
		var event PullRequestEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandlePullRequest(ctx, &event)
	case "installation":
		var event InstallationEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleInstallation(ctx, &event)
	case "installation_repositories":
		var event InstallationReposEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleInstallationRepos(ctx, &event)
	case "workflow_run":
		var event WorkflowRunEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleWorkflowRun(ctx, &event)
	case "deployment_status":
		var event DeploymentStatusEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleDeploymentStatus(ctx, &event)
	case "discussion":
		var event DiscussionEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleDiscussion(ctx, &event)
	case "dependabot_alert":
		var event DependabotAlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleDependabotAlert(ctx, &event)
	case "code_scanning_alert":
		var event CodeScanningAlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleCodeScanningAlert(ctx, &event)
	case "pull_request_review":
		var event PullRequestReviewEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandlePullRequestReview(ctx, &event)
	case "pull_request_review_comment":
		var event PullRequestReviewCommentEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandlePullRequestReviewComment(ctx, &event)
	case "security_advisory":
		var event SecurityAdvisoryEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleSecurityAdvisory(ctx, &event)
	case "issues":
		var event IssuesEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleIssues(ctx, &event)
	case "check_suite":
		var event CheckSuiteEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleCheckSuite(ctx, &event)
	case "check_run":
		var event CheckRunEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		dispatchErr = h.HandleCheckRun(ctx, &event)
	case "status":
		fmt.Printf("status event: state=%s\n", "received")
	case "installation_target", "meta":
		fmt.Printf("infrastructure event: %s\n", eventType)
	}

	if dispatchErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(true)
		_ = enc.Encode(map[string]string{"error": dispatchErr.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
