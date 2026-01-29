# GitHub Issue Link Component Integration Guide

## Overview

The `GitHubIssueLink` component provides a UI for linking Mattermost Boards cards to GitHub issues. It displays the GitHub connection status, allows searching for issues, and shows linked issue details.

## Component Location

- **Component**: `webapp/src/components/cardDetail/githubIssueLink.tsx`
- **Styles**: `webapp/src/components/cardDetail/githubIssueLink.scss`

## Features

1. **GitHub Connection Status**: Shows whether the user is connected to GitHub
2. **Issue Search**: Search for GitHub issues using the GitHub search API
3. **Link Display**: Shows linked issue with number, title, state, and labels
4. **Unlink Functionality**: Allows unlinking issues (when not readonly)

## Integration Example

To integrate the component into the card detail view, add it to `cardDetail.tsx`:

```tsx
import GitHubIssueLink from './githubIssueLink'

// Inside the CardDetail component, after CardRelations:
<CardRelations
    card={props.card}
    boardId={props.card.boardId}
    readonly={props.readonly || !canEditBoardCards}
    showCard={props.showCard}
/>

{/* GitHub Issue Link */}
<GitHubIssueLink
    card={props.card}
    readonly={props.readonly || !canEditBoardCards}
/>
```

## Props

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `card` | `Card` | Yes | The card object to link to GitHub issues |
| `readonly` | `boolean` | Yes | Whether the component is in readonly mode |

## Component States

### 1. Loading State
Shows a loading message while checking GitHub connection status.

### 2. Disconnected State
Displays a message prompting the user to connect their GitHub account.

### 3. Connected State
Shows the GitHub username and provides:
- Search interface for finding issues
- Display of linked issue (if any)
- Ability to link/unlink issues

## API Dependencies

The component depends on the following `octoClient` methods:

- `getGitHubConnected()`: Check if user is connected to GitHub
- `searchGitHubIssues(query)`: Search for GitHub issues

These methods are implemented in `webapp/src/octoClient.ts` and call the backend API endpoints:
- `GET /api/v2/github/connected`
- `GET /api/v2/github/issues?q={query}`

## Styling

The component uses BEM-style CSS classes with the `GitHubIssueLink` prefix. All styles follow the existing Mattermost Boards design patterns:

- Uses CSS variables for theming (e.g., `--center-channel-color-rgb`, `--button-bg-rgb`)
- Consistent spacing and border radius with other card detail components
- Responsive hover and focus states

## Future Enhancements

The current implementation provides the UI foundation. Future enhancements could include:

1. **Persistence**: Store linked issue information in card properties
2. **Sync Status**: Show if the linked issue has been updated
3. **Create Issue**: Add ability to create new GitHub issues directly from the card
4. **Multiple Links**: Support linking multiple issues to a single card
5. **Bidirectional Sync**: Sync card status with GitHub issue status

## Notes

- The component is read-only when the `readonly` prop is `true`
- Search is triggered by pressing Enter in the search input
- The component gracefully handles API errors with flash messages
- Issue labels are displayed with their GitHub colors

