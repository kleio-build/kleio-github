package kleiogithub

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkspaceLookup resolves a workspace from a GitHub App installation ID
// and links installations to workspaces on first install.
type WorkspaceLookup interface {
	FindByInstallationID(ctx context.Context, installationID int) (WorkspaceRef, error)
	// LinkInstallation associates a GitHub App installation ID with the
	// workspace that matches the given owner login. Called automatically
	// when the App is installed on an org or user account.
	LinkInstallation(ctx context.Context, installationID int, ownerLogin string) (WorkspaceRef, error)
}

// RepoStore manages repository records in a workspace.
type RepoStore interface {
	EnsureRepo(ctx context.Context, workspaceID string, repo RepoInfo) error
	EnsureRepoShort(ctx context.Context, workspaceID string, repo RepoShortInfo) error
}

// CaptureEmitter creates captures from GitHub events. The implementation
// lives in the proprietary kleio-app codebase and may route through
// the capture pipeline (embeddings, dedup, backlog synthesis).
type CaptureEmitter interface {
	EmitGitCommit(ctx context.Context, workspaceID string, commit CommitPayload) error
	EmitGitPR(ctx context.Context, workspaceID string, pr PRPayload) error
	EmitCIRun(ctx context.Context, workspaceID string, run CIRunPayload) error
	EmitDeployment(ctx context.Context, workspaceID string, deploy DeploymentPayload) error
	EmitDiscussion(ctx context.Context, workspaceID string, disc DiscussionPayload) error
	EmitSecurityAlert(ctx context.Context, workspaceID string, alert SecurityAlertPayload) error
	EmitPRReview(ctx context.Context, workspaceID string, review PRReviewPayload) error
	EmitPRReviewComment(ctx context.Context, workspaceID string, comment PRReviewCommentPayload) error
	EmitIssue(ctx context.Context, workspaceID string, issue IssuePayload) error
}

// CheckRunner runs the PR change analysis pipeline. The implementation
// lives in kleio-app/api/internal/services/check.
type CheckRunner interface {
	RunPRCheck(ctx context.Context, workspaceID string, repo string, prNumber int, headSHA, baseSHA string, installationID int) error
}

// PRDebouncer prevents rapid re-processing of the same PR on synchronize
// storms (e.g. force-push bursts).
type PRDebouncer struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
	window   time.Duration
}

func NewPRDebouncer(window time.Duration) *PRDebouncer {
	return &PRDebouncer{
		lastSeen: make(map[string]time.Time),
		window:   window,
	}
}

func (d *PRDebouncer) ShouldProcess(repo string, prNumber int) bool {
	key := fmt.Sprintf("%s#%d", repo, prNumber)
	now := time.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	if last, ok := d.lastSeen[key]; ok {
		if now.Sub(last) < d.window {
			return false
		}
	}
	d.lastSeen[key] = now
	return true
}
