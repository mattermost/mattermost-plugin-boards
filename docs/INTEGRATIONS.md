# Integrations

This document describes how to configure and use third-party integrations with the FamBear Boards plugin.

## Figma Integration

The FamBear Boards plugin supports integration with Figma through Personal Access Tokens.

### Setting up Figma Personal Access Token

#### 1. Generate a Figma Personal Access Token

To generate a Figma Personal Access Token:

1. Log in to your Figma account
2. Go to **Settings** → **Account** → **Personal Access Tokens**
3. Click **Generate new token**
4. Give your token a descriptive name (e.g., "Mattermost Boards Integration")
5. Copy the generated token (it starts with `figd_`)

**Important**: Save the token immediately - you won't be able to see it again after closing the dialog.

For detailed instructions, visit: https://help.figma.com/hc/en-us/articles/8085703771159-Manage-personal-access-tokens

#### 2. Configure the Token in Mattermost

1. Navigate to **System Console** → **Plugins** → **FamBear Boards**
2. Scroll down to the **Integrations** section
3. Paste your Figma Personal Access Token into the **Figma Personal Access Token** field
4. Click **Save**

### Security

The Figma Personal Access Token is stored securely in the Mattermost plugin configuration:

- **Storage**: The token is stored in the Mattermost database as part of the plugin settings
- **Encryption**: Mattermost encrypts plugin settings at rest using the database encryption
- **Access**: Only system administrators can view or modify the token through the System Console
- **Transmission**: The token is transmitted over HTTPS when saving settings

### Token Rotation

For security best practices, we recommend rotating your Figma Personal Access Token periodically:

1. Generate a new token in Figma (following the steps above)
2. Update the token in the System Console
3. Revoke the old token in Figma settings

### Accessing the Token from Plugin Code

For developers working on the plugin, the Figma token can be accessed from the plugin configuration:

#### Server-side (Go)

```go
// Get the current configuration
config := b.getConfiguration()

// Access the Figma token
figmaToken := config.FigmaPersonalAccessToken

// Use the token for Figma API calls
// Example: Make authenticated requests to Figma API
```

The token is available through the `configuration` struct in `server/boards/configuration.go`:

```go
type configuration struct {
    EnablePublicSharedBoards bool
    FigmaPersonalAccessToken string
}
```

#### Client-side (TypeScript/JavaScript)

The Figma token is **not** exposed to the client-side code for security reasons. All Figma API interactions should be performed server-side through plugin API endpoints.

If you need to interact with Figma from the client:

1. Create a server-side API endpoint that uses the token
2. Call that endpoint from the client-side code
3. The server will handle authentication with Figma using the stored token

### Troubleshooting

#### Token not working

- Verify the token is correctly copied (should start with `figd_`)
- Check that the token hasn't been revoked in Figma
- Ensure the token has the necessary permissions for your use case

#### Token not saved

- Verify you have system administrator permissions
- Check the Mattermost server logs for any errors
- Ensure the plugin is properly installed and activated

### Future Integrations

This Integrations section is designed to support additional third-party integrations in the future. Each integration will follow similar security practices for storing and accessing credentials.

