# GitHub PR Sync Integration

This document describes the integration between external PR sync scripts (like `poll-and-dispatch.sh`) and the Mattermost Boards plugin for tracking multiple GitHub Pull Requests per card.

## Overview

Cards can track multiple PRs with different statuses (e.g., one merged, one open after rework). The PR data is stored in a special card property and rendered by the `GitHubPRStatus` component.

## Property ID

The well-known property ID for GitHub PRs is:
```
agithubprs1prp7x9jkxd1ec66j
```

This property is automatically created on all boards and is managed by external sync scripts.

## Data Format

The property value must be a **JSON string** containing an **array of PR objects**. Each PR object has the following fields:

### Required Fields

- **`number`** (number): The PR number (e.g., `123`)
- **`title`** (string): The PR title
- **`url`** (string): Full GitHub PR URL (e.g., `https://github.com/owner/repo/pull/123`)
- **`status`** (string): One of the valid PR statuses (see below)
- **`repo`** (string): Repository identifier (e.g., `owner/repo`)

### Optional Fields

- **`branch`** (string): The source branch name (e.g., `feature/new-feature`)

### Valid Status Values

| Status | Description | Display Color | Category |
|--------|-------------|---------------|----------|
| `NEW` | Newly created PR | Gray | Active |
| `CI` | CI checks running | Blue | Active |
| `FAILED` | CI checks failed | Red | Active |
| `READY` | Ready for review/merge | Green | Active |
| `OPEN` | Generic open state | Green | Active |
| `MERGED` | PR was merged | Purple | Historical |
| `CLOSED` | PR was closed without merging | Red | Historical |

**Active PRs** are displayed prominently at the top with a colored left border.
**Historical PRs** are displayed below with muted styling.

## Example JSON Format

```json
[
  {
    "number": 123,
    "title": "Add new feature",
    "url": "https://github.com/owner/repo/pull/123",
    "status": "MERGED",
    "repo": "owner/repo",
    "branch": "feature/new-feature"
  },
  {
    "number": 456,
    "title": "Rework implementation",
    "url": "https://github.com/owner/repo/pull/456",
    "status": "OPEN",
    "repo": "owner/repo",
    "branch": "feature/new-feature-v2"
  }
]
```

## Integration with poll-and-dispatch.sh

The external `poll-and-dispatch.sh` script should:

1. **Query GitHub API** for ALL PRs matching the card code (from card title or properties)
   - Include open PRs (`state=open`)
   - Include closed PRs (`state=closed`)
   - Include merged PRs (check `merged` field)

2. **Map GitHub PR state to status**:
   - If `merged === true` → `"MERGED"`
   - If `state === "closed"` → `"CLOSED"`
   - If `state === "open"` → Determine based on CI status:
     - Check runs pending → `"CI"`
     - Check runs failed → `"FAILED"`
     - Check runs passed → `"READY"`
     - No checks or unknown → `"OPEN"`

3. **Build JSON array** with all PRs

4. **Update card property** using Mattermost API:
   ```bash
   # Example using curl
   curl -X PATCH "https://mm.example.com/plugins/focalboard/api/v2/boards/${BOARD_ID}/blocks/${CARD_ID}" \
     -H "Authorization: Bearer ${TOKEN}" \
     -H "Content-Type: application/json" \
     -d '{
       "updatedFields": {
         "properties": {
           "agithubprs1prp7x9jkxd1ec66j": "[{\"number\":123,\"title\":\"...\",\"url\":\"...\",\"status\":\"MERGED\",\"repo\":\"...\",\"branch\":\"...\"}]"
         }
       }
     }'
   ```

## Bot Status Logic

When determining if a card should be marked as "Done" based on PR status, the external script should implement the following logic:

### Set Card Status to "Done"

A card should be automatically set to "Done" **ONLY** when ALL of the following conditions are met:

1. **No active PRs exist**: No PR has status `NEW`, `CI`, `FAILED`, `READY`, or `OPEN`
2. **All PRs are completed**: Every PR is either `MERGED` or `CLOSED`
3. **At least one PR was merged**: There is at least one PR with status `MERGED` (indicating work was successfully completed)

### Keep Card Status as "In Progress" (or current status)

The card should remain in its current status (not automatically set to Done) if:

- **Any active work exists**: Any PR has status `NEW`, `CI`, `FAILED`, `READY`, or `OPEN`
- **No successful completion**: All PRs are `CLOSED` with none `MERGED` (work was abandoned)
- **No PRs exist**: The PR array is empty

### Example Logic (Pseudocode)

```bash
# Parse the PR array from the card property
prs=$(get_card_property "agithubprs1prp7x9jkxd1ec66j")

# Count PRs by status
active_count=$(count_prs_with_status "NEW|CI|FAILED|READY|OPEN")
merged_count=$(count_prs_with_status "MERGED")
closed_count=$(count_prs_with_status "CLOSED")
total_count=$(count_all_prs)

# Determine if card should be marked Done
if [ $active_count -eq 0 ] && [ $merged_count -gt 0 ] && [ $((merged_count + closed_count)) -eq $total_count ]; then
  # All PRs are complete (merged or closed) with at least one merge
  set_card_status "Done"
else
  # Keep current status (active work exists or no successful completion)
  # Do not change status
fi
```

## UI Rendering

The `GitHubPRStatus` component automatically:

1. Parses the JSON array from the property
2. Separates active and historical PRs
3. Displays active PRs first with prominent styling
4. Shows a divider if both active and historical PRs exist
5. Displays historical PRs below with muted styling
6. Shows branch names for each PR (if available)
7. Color-codes status badges based on the status value

## Security

- All URLs are sanitized to prevent XSS attacks
- Only `http://` and `https://` protocols are allowed
- Invalid JSON is silently ignored
- PRs with missing required fields are filtered out

