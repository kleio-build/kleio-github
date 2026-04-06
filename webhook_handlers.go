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

// HandleInstallation processes an installation event. On "created", it links
// the installation ID to the matching workspace and syncs the initial repos.
func (h *WebhookHandler) HandleInstallation(ctx context.Context, event *InstallationEvent) error {
	if event.Action != "created" {
		return nil
	}

	ownerLogin := event.Installation.Account.Login
	ws, err := h.workspace.LinkInstallation(ctx, event.Installation.ID, ownerLogin)
	if err != nil {
		fmt.Printf("installation link: no workspace for %q (installation %d): %v\n", ownerLogin, event.Installation.ID, err)
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

// HandleWorkflowRun processes a workflow_run event (CI).
func (h *WebhookHandler) HandleWorkflowRun(ctx context.Context, event *WorkflowRunEvent) error {
	if event.Action != "completed" {
		return nil
	}
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}
	return h.captures.EmitCIRun(ctx, ws.ID, CIRunPayload{
		RunID:        event.WorkflowRun.ID,
		Name:         event.WorkflowRun.Name,
		Conclusion:   event.WorkflowRun.Conclusion,
		HTMLURL:      event.WorkflowRun.HTMLURL,
		HeadSHA:      event.WorkflowRun.HeadSHA,
		HeadBranch:   event.WorkflowRun.HeadBranch,
		RunNumber:    event.WorkflowRun.RunNumber,
		ActorLogin:   event.WorkflowRun.Actor.Login,
		RepoFullName: event.Repository.FullName,
	})
}

// HandleDeploymentStatus processes a deployment_status event.
func (h *WebhookHandler) HandleDeploymentStatus(ctx context.Context, event *DeploymentStatusEvent) error {
	if event.Action != "created" {
		return nil
	}
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}
	return h.captures.EmitDeployment(ctx, ws.ID, DeploymentPayload{
		Environment:  event.Deployment.Environment,
		Status:       event.DeploymentStatus.State,
		Description:  event.DeploymentStatus.Description,
		HTMLURL:      event.DeploymentStatus.TargetURL,
		Ref:          event.Deployment.Ref,
		ActorLogin:   event.Deployment.Creator.Login,
		RepoFullName: event.Repository.FullName,
	})
}

// HandleDiscussion processes a discussion event.
func (h *WebhookHandler) HandleDiscussion(ctx context.Context, event *DiscussionEvent) error {
	if event.Action != "created" && event.Action != "answered" {
		return nil
	}
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}
	return h.captures.EmitDiscussion(ctx, ws.ID, DiscussionPayload{
		Number:       event.Discussion.Number,
		Title:        event.Discussion.Title,
		Body:         event.Discussion.Body,
		Category:     event.Discussion.Category.Name,
		HTMLURL:      event.Discussion.HTMLURL,
		AuthorLogin:  event.Discussion.User.Login,
		Action:       event.Action,
		RepoFullName: event.Repository.FullName,
	})
}

// HandleDependabotAlert processes a dependabot_alert event.
func (h *WebhookHandler) HandleDependabotAlert(ctx context.Context, event *DependabotAlertEvent) error {
	if event.Action != "created" {
		return nil
	}
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}
	return h.captures.EmitSecurityAlert(ctx, ws.ID, SecurityAlertPayload{
		AlertType:    "dependabot",
		Number:       event.Alert.Number,
		State:        event.Alert.State,
		Severity:     event.Alert.Severity,
		Summary:      event.Alert.Summary,
		HTMLURL:      event.Alert.HTMLURL,
		Package:      event.Alert.Package.Name,
		RepoFullName: event.Repository.FullName,
	})
}

// HandleCodeScanningAlert processes a code_scanning_alert event.
func (h *WebhookHandler) HandleCodeScanningAlert(ctx context.Context, event *CodeScanningAlertEvent) error {
	if event.Action != "created" {
		return nil
	}
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}
	return h.captures.EmitSecurityAlert(ctx, ws.ID, SecurityAlertPayload{
		AlertType:    "code_scanning",
		Number:       event.Alert.Number,
		State:        event.Alert.State,
		Severity:     event.Alert.Rule.Severity,
		Summary:      event.Alert.Rule.Description,
		HTMLURL:      event.Alert.HTMLURL,
		Rule:         event.Alert.Rule.Description,
		RepoFullName: event.Repository.FullName,
	})
}

// HandlePullRequestReview processes a pull_request_review event.
func (h *WebhookHandler) HandlePullRequestReview(ctx context.Context, event *PullRequestReviewEvent) error {
	if event.Action != "submitted" {
		return nil
	}
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}
	return h.captures.EmitPRReview(ctx, ws.ID, PRReviewPayload{
		PRNumber:      event.PullRequest.Number,
		PRTitle:       event.PullRequest.Title,
		ReviewState:   event.Review.State,
		ReviewBody:    event.Review.Body,
		ReviewURL:     event.Review.HTMLURL,
		ReviewerLogin: event.Review.User.Login,
		RepoFullName:  event.Repository.FullName,
	})
}

// HandleSecurityAdvisory processes a security_advisory event (global, no repo).
func (h *WebhookHandler) HandleSecurityAdvisory(ctx context.Context, event *SecurityAdvisoryEvent) error {
	if event.Action != "published" {
		return nil
	}
	// Security advisories are global (no installation/repo). Log but skip capture.
	fmt.Printf("security advisory published: %s — %s\n", event.Advisory.GHSAID, event.Advisory.Summary)
	return nil
}

// HandleIssues processes an issues event.
func (h *WebhookHandler) HandleIssues(ctx context.Context, event *IssuesEvent) error {
	if event.Action != "opened" && event.Action != "closed" {
		return nil
	}
	ws, err := h.resolveWorkspace(ctx, event.Installation)
	if err != nil {
		return fmt.Errorf("find workspace: %w", err)
	}
	var labels []string
	for _, l := range event.Issue.Labels {
		labels = append(labels, l.Name)
	}
	return h.captures.EmitIssue(ctx, ws.ID, IssuePayload{
		Number:       event.Issue.Number,
		Title:        event.Issue.Title,
		Body:         event.Issue.Body,
		State:        event.Issue.State,
		HTMLURL:      event.Issue.HTMLURL,
		AuthorLogin:  event.Issue.User.Login,
		Labels:       labels,
		Action:       event.Action,
		RepoFullName: event.Repository.FullName,
	})
}

// HandleCheckSuite logs check suite events (infrastructure for outbound checks).
func (h *WebhookHandler) HandleCheckSuite(ctx context.Context, event *CheckSuiteEvent) error {
	fmt.Printf("check_suite %s: %s (sha=%s)\n", event.Action, event.CheckSuite.Conclusion, event.CheckSuite.HeadSHA[:8])
	return nil
}

// HandleCheckRun logs check run events (infrastructure for outbound checks).
func (h *WebhookHandler) HandleCheckRun(ctx context.Context, event *CheckRunEvent) error {
	fmt.Printf("check_run %s: %s %s\n", event.Action, event.CheckRun.Name, event.CheckRun.Conclusion)
	return nil
}

func (h *WebhookHandler) resolveWorkspace(ctx context.Context, install *GHInstall) (WorkspaceRef, error) {
	if install == nil {
		return WorkspaceRef{}, fmt.Errorf("no installation in event")
	}
	ws, err := h.workspace.FindByInstallationID(ctx, install.ID)
	if err == nil {
		return ws, nil
	}
	// Fallback: the installation.created webhook may have been missed
	// (e.g. deployed after the app was installed). Try linking now using
	// the account login embedded in every webhook payload.
	if install.Account.Login != "" {
		ws, linkErr := h.workspace.LinkInstallation(ctx, install.ID, install.Account.Login)
		if linkErr == nil {
			fmt.Printf("auto-linked installation %d to workspace via %s (fallback)\n", install.ID, install.Account.Login)
			return ws, nil
		}
	}
	return WorkspaceRef{}, err
}
