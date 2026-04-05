package kleiogithub

import (
	"context"
	"fmt"
)

// HandlePush processes a push event: ensures the repo exists and emits
// a capture for each commit.
func (h *WebhookHandler) HandlePush(ctx context.Context, event *PushEvent) error {
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}

	err = h.repos.EnsureRepo(ctx, ws.ID, RepoInfo{
		GitHubRepoID:  event.Repository.ID,
		FullName:      event.Repository.FullName,
		HTMLURL:       event.Repository.HTMLURL,
		DefaultBranch: event.Repository.DefaultBranch,
	})
	if err != nil {
		return fmt.Errorf("ensure repo: %w", err)
	}

	for _, commit := range event.Commits {
		filesChanged := append(commit.Added, commit.Modified...)
		filesChanged = append(filesChanged, commit.Removed...)

		err := h.captures.EmitGitCommit(ctx, ws.ID, CommitPayload{
			SHA:          commit.ID,
			Message:      commit.Message,
			AuthorName:   commit.Author.Name,
			AuthorEmail:  commit.Author.Email,
			FilesChanged: filesChanged,
			RepoFullName: event.Repository.FullName,
			Timestamp:    commit.Timestamp,
		})
		if err != nil {
			fmt.Printf("failed to emit commit %s: %v\n", commit.ID[:8], err)
		}
	}

	return nil
}

// HandlePullRequest processes a pull_request event for opened/closed/reopened actions.
func (h *WebhookHandler) HandlePullRequest(ctx context.Context, event *PullRequestEvent) error {
	if event.Action != "opened" && event.Action != "closed" && event.Action != "reopened" {
		return nil
	}

	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}

	err = h.repos.EnsureRepo(ctx, ws.ID, RepoInfo{
		GitHubRepoID:  event.Repository.ID,
		FullName:      event.Repository.FullName,
		HTMLURL:       event.Repository.HTMLURL,
		DefaultBranch: event.Repository.DefaultBranch,
	})
	if err != nil {
		return fmt.Errorf("ensure repo: %w", err)
	}

	return h.captures.EmitGitPR(ctx, ws.ID, PRPayload{
		Number:       event.PullRequest.Number,
		Title:        event.PullRequest.Title,
		Body:         event.PullRequest.Body,
		State:        event.PullRequest.State,
		Merged:       event.PullRequest.Merged,
		HTMLURL:      event.PullRequest.HTMLURL,
		AuthorLogin:  event.PullRequest.User.Login,
		RepoFullName: event.Repository.FullName,
		Action:       event.Action,
	})
}

// HandleInstallation processes an installation event (repos added on first install).
func (h *WebhookHandler) HandleInstallation(ctx context.Context, event *InstallationEvent) error {
	if event.Action != "created" {
		return nil
	}

	ws, err := h.workspace.FindByInstallationID(ctx, event.Installation.ID)
	if err != nil {
		return nil
	}

	for _, r := range event.Repositories {
		h.repos.EnsureRepoShort(ctx, ws.ID, RepoShortInfo{
			ID:       r.ID,
			FullName: r.FullName,
		})
	}
	return nil
}

// HandleInstallationRepos processes repos added/removed from an existing installation.
func (h *WebhookHandler) HandleInstallationRepos(ctx context.Context, event *InstallationReposEvent) error {
	ws, err := h.workspace.FindByInstallationID(ctx, event.Installation.ID)
	if err != nil {
		return nil
	}

	for _, r := range event.RepositoriesAdded {
		h.repos.EnsureRepoShort(ctx, ws.ID, RepoShortInfo{
			ID:       r.ID,
			FullName: r.FullName,
		})
	}
	return nil
}

func (h *WebhookHandler) resolveWorkspace(ctx context.Context, install *GHInstall) (WorkspaceRef, error) {
	if install == nil {
		return WorkspaceRef{}, fmt.Errorf("no installation in event")
	}
	return h.workspace.FindByInstallationID(ctx, install.ID)
}
