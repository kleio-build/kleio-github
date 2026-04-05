package kleiogithub

import "context"

// WorkspaceLookup resolves a workspace from a GitHub App installation ID.
type WorkspaceLookup interface {
	FindByInstallationID(ctx context.Context, installationID int) (WorkspaceRef, error)
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
}
