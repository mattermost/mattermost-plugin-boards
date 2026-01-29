// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// GitHub integration types
// These types match the server-side types in server/services/github/types.go

export interface GitHubRepository {
    id: number
    name: string
    full_name: string
    owner: string
    private: boolean
    html_url: string
    description: string
    default_branch: string
}

export interface GitHubUser {
    login: string
    id: number
    avatar_url: string
    html_url: string
}

export interface GitHubLabel {
    id: number
    name: string
    color: string
    description: string
}

export interface GitHubMilestone {
    number: number
    title: string
    state: string
}

export interface GitHubIssue {
    number: number
    title: string
    body: string
    state: string
    html_url: string
    created_at: string
    updated_at: string
    user: GitHubUser
    labels: GitHubLabel[]
    assignees: GitHubUser[]
    milestone?: GitHubMilestone
}

export interface CreateGitHubIssueRequest {
    owner: string
    repo: string
    title: string
    body: string
    labels?: string[]
    assignees?: string[]
    milestone?: number
}

export interface GitHubConnectedResponse {
    connected: boolean
    github_username?: string
}

