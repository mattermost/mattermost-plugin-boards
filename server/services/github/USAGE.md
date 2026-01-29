# GitHub Service Usage Guide

## Overview

The GitHub service provides IPC (Inter-Plugin Communication) functionality to interact with the Mattermost GitHub plugin. It allows the Boards plugin to check user connections, manage repositories, create and search issues, and retrieve pull request details.

## Initialization

```go
import (
    "github.com/mattermost/mattermost-plugin-boards/server/services/github"
    "github.com/mattermost/mattermost-plugin-boards/server/model"
)

// Create a new GitHub service instance
// The ServicesAPI interface is typically provided by the plugin adapter
githubService := github.New(servicesAPI)
```

## Usage Examples

### 1. Check if User is Connected to GitHub

```go
userID := "user123"
connected, err := githubService.IsUserConnected(userID)
if err != nil {
    // Handle error
    log.Error("Failed to check GitHub connection", err)
    return
}

if connected {
    log.Info("User is connected to GitHub")
} else {
    log.Info("User is not connected to GitHub")
}
```

### 2. Get User's Repositories

```go
// Get all repositories for a user
repos, err := githubService.GetRepositories(userID, "")
if err != nil {
    log.Error("Failed to get repositories", err)
    return
}

// Get repositories for a specific channel
repos, err := githubService.GetRepositories(userID, "channel123")
if err != nil {
    log.Error("Failed to get repositories", err)
    return
}

for _, repo := range repos {
    fmt.Printf("Repository: %s (%s)\n", repo.FullName, repo.HTMLURL)
}
```

### 3. Create a GitHub Issue

```go
createReq := github.CreateIssueRequest{
    Owner:     "mattermost",
    Repo:      "mattermost-plugin-boards",
    Title:     "Bug: Issue title",
    Body:      "Detailed description of the issue",
    Labels:    []string{"bug", "high-priority"},
    Assignees: []string{"username1", "username2"},
    Milestone: 5,
}

issue, err := githubService.CreateIssue(userID, createReq)
if err != nil {
    log.Error("Failed to create issue", err)
    return
}

fmt.Printf("Created issue #%d: %s\n", issue.Number, issue.HTMLURL)
```

### 4. Get Issue Details

```go
issue, err := githubService.GetIssue(userID, "mattermost", "mattermost-plugin-boards", 42)
if err != nil {
    log.Error("Failed to get issue", err)
    return
}

fmt.Printf("Issue #%d: %s\n", issue.Number, issue.Title)
fmt.Printf("State: %s\n", issue.State)
fmt.Printf("URL: %s\n", issue.HTMLURL)
```

### 5. Search Issues

```go
issues, err := githubService.SearchIssues(userID, "is:open label:bug")
if err != nil {
    log.Error("Failed to search issues", err)
    return
}

fmt.Printf("Found %d issues\n", len(issues))
for _, issue := range issues {
    fmt.Printf("  #%d: %s\n", issue.Number, issue.Title)
}
```

### 6. Get Pull Request Details

```go
pr, err := githubService.GetPRDetails(userID, "mattermost", "mattermost-plugin-boards", 10)
if err != nil {
    log.Error("Failed to get PR details", err)
    return
}

fmt.Printf("PR #%d: %s\n", pr.Number, pr.Title)
fmt.Printf("State: %s\n", pr.State)
fmt.Printf("Mergeable: %v\n", pr.Mergeable)
fmt.Printf("Merged: %v\n", pr.Merged)
fmt.Printf("Head: %s (%s)\n", pr.Head.Ref, pr.Head.SHA)
fmt.Printf("Base: %s (%s)\n", pr.Base.Ref, pr.Base.SHA)
```

## Error Handling

All methods return errors that should be handled appropriately. Common error scenarios include:

- User not connected to GitHub (401 Unauthorized)
- Resource not found (404 Not Found)
- Validation errors (400 Bad Request)
- GitHub plugin not available (no response)

Example error handling:

```go
issue, err := githubService.GetIssue(userID, owner, repo, number)
if err != nil {
    if strings.Contains(err.Error(), "not_found") {
        // Handle not found case
        return fmt.Errorf("issue not found")
    }
    if strings.Contains(err.Error(), "not_connected") {
        // Handle user not connected case
        return fmt.Errorf("user must connect to GitHub first")
    }
    // Handle other errors
    return err
}
```

## Integration with ServicesAPI

The GitHub service requires a `ServicesAPI` interface that provides:

- `PluginHTTP(req *http.Request) *http.Response` - For IPC communication
- `GetLogger() mlog.LoggerIFace` - For logging

This interface is typically satisfied by the plugin's API adapter (`model.ServicesAPI`).

## Notes

- All methods require a valid `userID` parameter
- The service uses the `Mattermost-User-ID` header for user authentication
- IPC calls are made to the GitHub plugin's API endpoints
- The GitHub plugin must be installed and active for this service to work

