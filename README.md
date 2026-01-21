# Mattermost Boards

### A self-hosted plugin for managing boards within a Mattermost installation

Mattermost boards plugins is an open source, multilingual, self-hosted project management tool that's an alternative to Trello, Notion, and Asana.

## Try Mattermost Boards Plugin 

Access the latest releases of the mattermost boards plugin by downloading the `mattermost-plugin-focalboard.tar.gz` file from the releases in this repository: <https://github.com/mattermost/mattermost-plugin-boards/releases>. After downloading and installing the plugin in the System Console, select the menu in the top left corner and select **Boards**. 

### Getting started

Clone [mattermost](https://github.com/mattermost/mattermost-server) into sibling directory.

You also want to have the environment variable `MM_DEBUG"true"` set, otherwise the plugin
will be compiled for Linux, Windows, and Darwin ARM64 and x64 architecture every single time. Setting
the `MM_DEBUG` to `true` makes the plugin compile and build only for the OS and architecture
you are building on.

In your Mattermost configuration file, ensure that `PluginSettings.EnableUploads` is set to `true`, and `FileSettings.MaxFileSize` is
set to a large enough value to accept the plugin bundle (eg `256000000`).

### Installing Dependencies 

```sh
cd ./webapp
npm install
```

### Building the plugin

Run the following command in the plugin repository to prepare a compiled, distributable plugin ZIP file:

```bash
make dist
```

After a successful build, a `.tar.gz` file in the `/dist` folder will be created which can be uploaded to Mattermost. To avoid having to manually install your plugin, deploy your plugin using one of the following options.

##### Building in Dev Mode

Set the following environment variables to true before running `make dist`-

1. MM_DEBUG

### Deploying with Local Mode

If your Mattermost server is running locally, you can
enable [local mode](https://docs.mattermost.com/manage/mmctl-command-line-tool.html) to streamline deploying
your plugin. Edit your server configuration as follows:

```
{
    "ServiceSettings": {
        ...
        "EnableLocalMode": true,
        "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket"
     }
}
```

and then deploy your plugin:

```bash
make deploy
```

If developing a plugin with a web app, watch for changes and deploy those automatically:

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
make watch-plugin
```

## How to Release

### Automated Release (Recommended)

This project uses GitHub Actions for automated releases:

1. **Update version in `plugin.json`**
   ```bash
   # Edit plugin.json and change the "version" field
   # Example: "version": "9.2.3"
   ```

2. **Commit and push to release branch**
   ```bash
   git add plugin.json
   git commit -m "Bump version to 9.2.3"
   git push origin main:release
   ```

3. **GitHub Actions will automatically:**
   - Build the plugin for all platforms (Linux, macOS, Windows)
   - Create a git tag `v{version}`
   - Create a GitHub Release
   - Upload universal bundle `boards-{version}.tar.gz` (~150-160 MB)
   - **Enable automatic updates through Mattermost UI**

For detailed instructions, see:
- [RELEASE.md](RELEASE.md) - Complete release guide
- [QUICKSTART-RELEASE.md](QUICKSTART-RELEASE.md) - Quick start
- [docs/AUTO-UPDATE-GUIDE.md](docs/AUTO-UPDATE-GUIDE.md) - Auto-update setup

### Local Build

To build the release locally:

**Linux/macOS:**
```bash
./scripts/build-release.sh
```

**Windows:**
```powershell
.\scripts\build-release.ps1
```

Or manually:
```bash
make dist-linux
```


### Unit testing

Before checking in commits, run `make ci`, which is similar to the `.gitlab-ci.yml` workflow and includes:

* **Server unit tests**: `make server-test`
* **Web app ESLint**: `cd webapp; npm run check`
* **Web app unit tests**: `cd webapp; npm run test`
