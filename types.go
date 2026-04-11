package kleiogithub

// User represents a GitHub user profile returned from the /user API.
type User struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// Org represents a GitHub organization the authenticated user belongs to.
type Org struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

// RepoOwner identifies a repository's owner on GitHub.
type RepoOwner struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// Repo represents a GitHub repository.
type Repo struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	DefaultBranch string    `json:"default_branch"`
	HTMLURL       string    `json:"html_url"`
	Private       bool      `json:"private"`
	Description   string    `json:"description"`
	Owner         RepoOwner `json:"owner"`
	Fork          bool      `json:"fork"`
}

// WorkspaceRef is a minimal reference to a workspace, returned by WorkspaceLookup.
type WorkspaceRef struct {
	ID string
}

// RepoInfo carries the data needed to ensure a repository exists in a workspace.
type RepoInfo struct {
	GitHubRepoID  int
	FullName      string
	HTMLURL       string
	DefaultBranch string
}

// CommitPayload is the data extracted from a push event commit.
type CommitPayload struct {
	SHA          string
	Message      string
	AuthorName   string
	AuthorEmail  string
	FilesChanged []string
	RepoFullName string
	Timestamp    string
}

// PRPayload is the data extracted from a pull_request event.
type PRPayload struct {
	Number       int
	Title        string
	Body         string
	State        string
	Merged       bool
	HTMLURL      string
	AuthorLogin  string
	RepoFullName string
	Action       string
}

// RepoShortInfo is a minimal repo reference from installation events.
type RepoShortInfo struct {
	ID       int
	FullName string
}

// PushEvent is the GitHub webhook push event payload.
type PushEvent struct {
	Ref          string        `json:"ref"`
	Repository   GHRepository  `json:"repository"`
	Commits      []GHCommit    `json:"commits"`
	Installation *GHInstall    `json:"installation"`
}

// GHRepository is the repository object in webhook payloads.
type GHRepository struct {
	ID            int    `json:"id"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	DefaultBranch string `json:"default_branch"`
}

// GHCommit is a commit in a push event.
type GHCommit struct {
	ID        string   `json:"id"`
	Message   string   `json:"message"`
	Author    GHAuthor `json:"author"`
	Added     []string `json:"added"`
	Modified  []string `json:"modified"`
	Removed   []string `json:"removed"`
	Timestamp string   `json:"timestamp"`
}

// GHAuthor is a commit author.
type GHAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GHInstall is the installation reference in webhook payloads.
type GHInstall struct {
	ID      int            `json:"id"`
	Account GHInstallOwner `json:"account"`
}

// GHInstallOwner is the account (org or user) that owns an installation.
type GHInstallOwner struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// PullRequestEvent is the GitHub webhook pull_request event payload.
type PullRequestEvent struct {
	Action       string         `json:"action"`
	Number       int            `json:"number"`
	PullRequest  GHPullRequest  `json:"pull_request"`
	Repository   GHRepository   `json:"repository"`
	Installation *GHInstall     `json:"installation"`
}

// GHPullRequest is the pull request object in webhook payloads.
type GHPullRequest struct {
	ID      int    `json:"id"`
	Number  int    `json:"number"`
	State   string `json:"state"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Merged  bool   `json:"merged"`
	HTMLURL string `json:"html_url"`
	User    GHUser `json:"user"`
}

// GHUser is a GitHub user in webhook payloads.
type GHUser struct {
	Login string `json:"login"`
}

// InstallationEvent is the GitHub webhook installation event payload.
type InstallationEvent struct {
	Action       string        `json:"action"`
	Installation GHInstall     `json:"installation"`
	Repositories []GHRepoShort `json:"repositories"`
}

// InstallationReposEvent is the GitHub webhook installation_repositories event payload.
type InstallationReposEvent struct {
	Action              string        `json:"action"`
	Installation        GHInstall     `json:"installation"`
	RepositoriesAdded   []GHRepoShort `json:"repositories_added"`
	RepositoriesRemoved []GHRepoShort `json:"repositories_removed"`
}

// GHRepoShort is a minimal repo reference in installation events.
type GHRepoShort struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
}

// ---------------------------------------------------------------------------
// New signal-producing event types
// ---------------------------------------------------------------------------

type WorkflowRunEvent struct {
	Action       string       `json:"action"`
	WorkflowRun  GHWorkflowRun `json:"workflow_run"`
	Repository   GHRepository `json:"repository"`
	Installation *GHInstall   `json:"installation"`
}

type GHWorkflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
	HeadSHA    string `json:"head_sha"`
	HeadBranch string `json:"head_branch"`
	RunNumber  int    `json:"run_number"`
	Actor      GHUser `json:"actor"`
}

type DeploymentStatusEvent struct {
	Action           string             `json:"action"`
	DeploymentStatus GHDeploymentStatus `json:"deployment_status"`
	Deployment       GHDeployment       `json:"deployment"`
	Repository       GHRepository       `json:"repository"`
	Installation     *GHInstall         `json:"installation"`
}

type GHDeployment struct {
	ID          int64  `json:"id"`
	Ref         string `json:"ref"`
	Environment string `json:"environment"`
	Creator     GHUser `json:"creator"`
}

type GHDeploymentStatus struct {
	ID          int64  `json:"id"`
	State       string `json:"state"`
	Description string `json:"description"`
	TargetURL   string `json:"target_url"`
}

type DiscussionEvent struct {
	Action       string       `json:"action"`
	Discussion   GHDiscussion `json:"discussion"`
	Repository   GHRepository `json:"repository"`
	Installation *GHInstall   `json:"installation"`
}

type GHDiscussion struct {
	Number   int                  `json:"number"`
	Title    string               `json:"title"`
	Body     string               `json:"body"`
	HTMLURL  string               `json:"html_url"`
	User     GHUser               `json:"user"`
	Category GHDiscussionCategory `json:"category"`
}

type GHDiscussionCategory struct {
	Name string `json:"name"`
}

type DependabotAlertEvent struct {
	Action       string            `json:"action"`
	Alert        GHDependabotAlert `json:"alert"`
	Repository   GHRepository      `json:"repository"`
	Installation *GHInstall        `json:"installation"`
}

type GHDependabotAlert struct {
	Number   int                    `json:"number"`
	State    string                 `json:"state"`
	Summary  string                 `json:"summary"`
	Severity string                 `json:"severity"`
	HTMLURL  string                 `json:"html_url"`
	Package  GHDependabotPackage    `json:"dependency"`
}

type GHDependabotPackage struct {
	Name string `json:"name"`
}

type CodeScanningAlertEvent struct {
	Action       string               `json:"action"`
	Alert        GHCodeScanningAlert  `json:"alert"`
	Repository   GHRepository         `json:"repository"`
	Installation *GHInstall           `json:"installation"`
}

type GHCodeScanningAlert struct {
	Number  int                `json:"number"`
	State   string             `json:"state"`
	Rule    GHCodeScanningRule `json:"rule"`
	HTMLURL string             `json:"html_url"`
}

type GHCodeScanningRule struct {
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type PullRequestReviewEvent struct {
	Action       string        `json:"action"`
	Review       GHPRReview    `json:"review"`
	PullRequest  GHPullRequest `json:"pull_request"`
	Repository   GHRepository  `json:"repository"`
	Installation *GHInstall    `json:"installation"`
}

type GHPRReview struct {
	ID      int64  `json:"id"`
	State   string `json:"state"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	User    GHUser `json:"user"`
}

type PullRequestReviewCommentEvent struct {
	Action       string                `json:"action"`
	Comment      GHPRReviewComment     `json:"comment"`
	PullRequest  GHPullRequest         `json:"pull_request"`
	Repository   GHRepository          `json:"repository"`
	Installation *GHInstall            `json:"installation"`
}

type GHPRReviewComment struct {
	ID       int64  `json:"id"`
	Body     string `json:"body"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	DiffHunk string `json:"diff_hunk"`
	HTMLURL  string `json:"html_url"`
	User     GHUser `json:"user"`
}

type SecurityAdvisoryEvent struct {
	Action   string             `json:"action"`
	Advisory GHSecurityAdvisory `json:"security_advisory"`
}

type GHSecurityAdvisory struct {
	GHSAID   string `json:"ghsa_id"`
	Summary  string `json:"summary"`
	Severity string `json:"severity"`
	HTMLURL  string `json:"html_url"`
}

type IssuesEvent struct {
	Action       string       `json:"action"`
	Issue        GHIssue      `json:"issue"`
	Repository   GHRepository `json:"repository"`
	Installation *GHInstall   `json:"installation"`
}

type GHIssue struct {
	Number  int       `json:"number"`
	Title   string    `json:"title"`
	Body    string    `json:"body"`
	State   string    `json:"state"`
	HTMLURL string    `json:"html_url"`
	User    GHUser    `json:"user"`
	Labels  []GHLabel `json:"labels"`
}

type GHLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ---------------------------------------------------------------------------
// Context/infrastructure events (logged, no captures in v0)
// ---------------------------------------------------------------------------

type CheckSuiteEvent struct {
	Action       string       `json:"action"`
	CheckSuite   GHCheckSuite `json:"check_suite"`
	Repository   GHRepository `json:"repository"`
	Installation *GHInstall   `json:"installation"`
}

type GHCheckSuite struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HeadSHA    string `json:"head_sha"`
}

type CheckRunEvent struct {
	Action       string       `json:"action"`
	CheckRun     GHCheckRun   `json:"check_run"`
	Repository   GHRepository `json:"repository"`
	Installation *GHInstall   `json:"installation"`
}

type GHCheckRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// ---------------------------------------------------------------------------
// New payload types for emitter interface
// ---------------------------------------------------------------------------

type CIRunPayload struct {
	RunID        int64
	Name         string
	Conclusion   string
	HTMLURL      string
	HeadSHA      string
	HeadBranch   string
	RunNumber    int
	ActorLogin   string
	RepoFullName string
}

type DeploymentPayload struct {
	Environment  string
	Status       string
	Description  string
	HTMLURL      string
	Ref          string
	ActorLogin   string
	RepoFullName string
}

type DiscussionPayload struct {
	Number       int
	Title        string
	Body         string
	Category     string
	HTMLURL      string
	AuthorLogin  string
	Action       string
	RepoFullName string
}

type SecurityAlertPayload struct {
	AlertType    string
	Number       int
	State        string
	Severity     string
	Summary      string
	HTMLURL      string
	Package      string
	Rule         string
	RepoFullName string
}

type PRReviewPayload struct {
	PRNumber      int
	PRTitle       string
	ReviewState   string
	ReviewBody    string
	ReviewURL     string
	ReviewerLogin string
	RepoFullName  string
}

type PRReviewCommentPayload struct {
	PRNumber      int
	PRTitle       string
	CommentID     int64
	Body          string
	Path          string
	Line          int
	DiffHunk      string
	HTMLURL       string
	AuthorLogin   string
	AuthorType    string // "User" or "Bot"
	RepoFullName  string
}

type IssuePayload struct {
	Number       int
	Title        string
	Body         string
	State        string
	HTMLURL      string
	AuthorLogin  string
	Labels       []string
	Action       string
	RepoFullName string
}
