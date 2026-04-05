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
	ID int `json:"id"`
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
