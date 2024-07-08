> [!WARNING]
> **Effective September 15th, 2023, Mattermost, Inc. staff are no longer reviewing or merging pull requests for either Focalboard or the Mattermost Boards plugin in this repository (`mattermost/focalboard`). We encourage the community to fork this repository for continued development and contributions.**
>
> The reason behind these changes is to focus Mattermost developer resources on improving the platform’s performance and core features to ensure Mattermost continues being resilient, stable, and best-in-breed for critical operations.
>
> ️💡 [Learn more](https://forum.mattermost.com/t/upcoming-product-changes-to-boards-and-various-plugins/16669)

# Mattermost boards plugin 

### A self-hosted plugin for managing boards withing a Mattermost installation

Focalboard is an open source, multilingual, self-hosted project management tool that's an alternative to Trello, Notion, and Asana.

**[Mattermost Boards](https://mattermost.com/boards/)** is the Mattermost plugin version of Focalboard that combines project management tools with messaging and collaboration for teams of all sizes. To access and use **Mattermost Boards**, install or upgrade to Mattermost v6.0 or later as a [self-hosted server](https://docs.mattermost.com/guides/deployment.html?utm_source=focalboard&utm_campaign=focalboard) or [Cloud server](https://mattermost.com/get-started/?utm_source=focalboard&utm_campaign=focalboard). After logging into Mattermost, select the menu in the top left corner of Mattermost and select **Boards**.

![Focalboard](website/site/static/img/hero.jpg)

It helps define, organize, track and manage work across individuals and teams. Focalboard comes in three editions:

* **[Focalboard plugin](https://github.com/mattermost/focalboard/releases)**: The Focalboard plugin integrates into an exsting Mattermost instance to combine project management tools with messaging and collaboration for teams of all sizes.

* **[Personal Desktop](https://www.focalboard.com/docs/personal-edition/desktop/)**: A standalone, single-user [macOS](https://apps.apple.com/app/apple-store/id1556908618?pt=2114704&ct=website&mt=8), [Windows](https://www.microsoft.com/store/apps/9NLN2T0SX9VF?cid=website), or [Linux](https://www.focalboard.com/download/personal-edition/desktop/#linux-desktop) desktop app for your own todos and personal projects.

* **[Personal Server](https://www.focalboard.com/download/personal-edition/ubuntu/)**: A standalone, multi-user server for development and personal use.

## Try Focalboard

### Mattermost Plugin

After downloading and installing the plugin in the System Console, select the menu in the top left corner and select **Boards**. Access the latest releases of the focalboard plugin by downloading the `mattermost-plugin-focalboard.tar.gz` file from the releases in this repository: <https://github.com/mattermost/focalboard/releases>


### API Docs

Boards API docs can be found over at <https://htmlpreview.github.io/?https://github.com/mattermost/focalboard/blob/main/server/swagger/docs/html/index.html>

### Getting started

Our [developer guide](https://developers.mattermost.com/contribute/more-info/focalboard/mattermost-boards-setup-guide/) has detailed instructions on how to set up your development environment for the **Mattermost boards plugin**. You can also join the [~Focalboard community channel](https://community.mattermost.com/core/channels/focalboard) to connect with other developers.

Clone [mattermost-server](https://github.com/mattermost/mattermost-server) into sibling directory.

You also want to have the environment variable `MM_DEBUG"true"` set, otherwise the plugin
will be compiled for Linux, Windows, and Darwin ARM64 and x64 architecture every single time. Setting
the `MM_DEBUG` to `true` makes the plugin compile and build only for the OS and architecture
you are building on.

In your Mattermost configuration file, ensure that `PluginSettings.EnableUploads` is set to `true`, and `FileSettings.MaxFileSize` is
set to a large enough value to accept the plugin bundle (eg `256000000`).

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
enable [local mode](https://docs.mattermost.com/administration/mmctl-cli-tool.html#local-mode) to streamline deploying
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



### Unit testing

Before checking in commits, run `make ci`, which is similar to the `.gitlab-ci.yml` workflow and includes:

* **Server unit tests**: `make server-test`
* **Web app ESLint**: `cd webapp; npm run check`
* **Web app unit tests**: `cd webapp; npm run test`

### Staying informed

* **Changes**: See the [CHANGELOG](CHANGELOG.md) for the latest updates
* **Bug Reports**: [File a bug report](https://github.com/mattermost/focalboard/issues/new?assignees=&labels=bug&template=bug_report.md&title=)
* **Chat**: Join the [~Focalboard community channel](https://community.mattermost.com/core/channels/focalboard)
