# kleio-github

Open-source GitHub integration for [Kleio](https://kleio.build) — the engineering provenance platform.

This module handles all interaction between Kleio and GitHub:

- **OAuth / GitHub App user authorization** — login flow for the Kleio web app
- **Webhook processing** — inbound `push`, `pull_request`, `installation`, and `installation_repositories` events
- **Installation token management** — scoped, short-lived tokens for GitHub App API calls via a dedicated signer service
- **PR comments and check runs** — outbound GitHub API calls (coming soon)

## Why is this open source?

Kleio requests access to your repositories via a GitHub App. We believe you should be able to audit exactly what we do with that access. This module is the complete integration layer — what data we read, what we store, and what we post back.

Kleio's competitive advantage is the capture pipeline, AI synthesis, and product UX — not how we parse `push` events.

## Permissions

The Kleio GitHub App requests these permissions:

### Account permissions

| Permission | Access | Used for |
|------------|--------|----------|
| Email addresses | Read | User login, profile |
| Organization members | Read | Workspace discovery |

### Repository permissions

| Permission | Access | Used for |
|------------|--------|----------|
| Pull requests | Read & write | PR context comments |
| Checks | Read & write | Decision traceability checks |
| Issues | Read | Linking captures to issues |
| Contents | Read | File context for captures |
| Metadata | Read | Required base permission |

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌────────────────────┐
│  GitHub.com  │────>│  kleio-github    │────>│  kleio-app         │
│              │     │  (this module)   │     │  (proprietary)     │
│  webhooks    │     │                  │     │                    │
│  OAuth       │     │  - parse events  │     │  - capture pipeline│
│  API         │     │  - verify sigs   │     │  - embeddings/AI   │
│              │     │  - auth flow     │     │  - database        │
└─────────────┘     └──────────────────┘     └────────────────────┘
                              │
                              │ scoped token requests
                              ▼
                    ┌──────────────────┐
                    │ kleio-github-    │
                    │ signer           │
                    │ (holds PEM key)  │
                    └──────────────────┘
```

The main Kleio application never holds the GitHub App private key. A dedicated, minimal signer service mints scoped installation tokens on request.

## Usage

```go
import gh "github.com/kleio-build/kleio-github"

// Auth client for OAuth / GitHub App user authorization
auth := gh.NewAuthClient(gh.AuthConfig{
    ClientID:     os.Getenv("GITHUB_APP_CLIENT_ID"),
    ClientSecret: os.Getenv("GITHUB_APP_CLIENT_SECRET"),
    RedirectURL:  "https://app.kleio.build/auth/github/callback",
})

// Webhook handler — inject your own implementations
handler := gh.NewWebhookHandler(
    os.Getenv("GITHUB_WEBHOOK_SECRET"),
    myWorkspaceLookup,  // implements gh.WorkspaceLookup
    myRepoStore,        // implements gh.RepoStore
    myCaptureEmitter,   // implements gh.CaptureEmitter
)

// GitHub App service for installation tokens
app := gh.NewAppService(gh.AppConfig{
    AppID:     12345,
    SignerURL: os.Getenv("GITHUB_SIGNER_URL"),
})
```

## Interfaces

The module defines three interfaces that consumers must implement:

- **`WorkspaceLookup`** — resolves a workspace from a GitHub installation ID
- **`RepoStore`** — ensures repositories exist in a workspace
- **`CaptureEmitter`** — creates captures from git commits and pull requests

These keep database and business logic out of the public module.

## License

MIT
