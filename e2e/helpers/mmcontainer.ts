import {File as NodeFile} from 'buffer';
import type {StartedTestContainer, StartedNetwork} from 'testcontainers';
import type {StartedPostgreSqlContainer} from '@testcontainers/postgresql';
import {Client4} from "@mattermost/client";
import { Client } from 'pg'
import { pluginAdminConfigApiFromClient } from './plugin-http';

if (typeof globalThis.File === 'undefined') {
    Object.assign(globalThis, {File: NodeFile});
}

const {GenericContainer, Network, Wait} = require('testcontainers') as typeof import('testcontainers');
const {PostgreSqlContainer} = require('@testcontainers/postgresql') as typeof import('@testcontainers/postgresql');

const defaultEmail           = "admin@example.com";
const defaultUsername        = "admin";
const defaultPassword        = "admin";
const defaultTeamName        = "test";
const defaultTeamDisplayName = "Test";
const defaultMattermostImage = "mattermost/mattermost-enterprise-edition:11.5.1";

type PluginConfig = Record<string, unknown>;
type PluginConfigInput = PluginConfig | {config: PluginConfig};

// MattermostContainer represents the mattermost container type used in the module
export default class MattermostContainer {
    container: StartedTestContainer;
    pgContainer: StartedPostgreSqlContainer;
    network:     StartedNetwork;
    email: string;
    username:    string;
	password:    string;
    teamName: string;
    teamDisplayName: string;
    envs:        {[key: string]: string};
    command:    string[];
    configFile: any[];
    plugins: any[];
    private logStream: any;
    private isLogStreamClosed: boolean;

    url(): string {
        const containerPort = this.container.getMappedPort(8065)
        const host = this.container.getHost()
        return `http://${host}:${containerPort}`
    }

    db = async (): Promise<Client> => {
        const port = this.pgContainer.getMappedPort(5432)
        const host = this.pgContainer.getHost()
        const database = "mattermost_test"
        const client = new Client({user: "user", password: "pass", host, port, database})
        await client.connect()
        return client
    }

    getAdminClient = async (): Promise<Client4> => {
        return this.getClient(this.username, this.password)
    }

    getClient = async (username: string, password: string): Promise<Client4> => {
        const url = this.url()
        const client = new Client4()
        client.setUrl(url)
        await client.login(username, password)
        return client
    }

    stop = async () => {
        let firstError: unknown;

        if (this.pgContainer) {
            try {
                await this.pgContainer.stop()
            } catch (err) {
                firstError ??= err;
            }
        }
        if (this.container) {
            try {
                await this.container.stop()
            } catch (err) {
                firstError ??= err;
            }
        }
        if (this.network) {
            try {
                await this.network.stop()
            } catch (err) {
                firstError ??= err;
            }
        }
        if (this.logStream && !this.logStream.destroyed && !this.logStream.writableEnded) {
            try {
                this.isLogStreamClosed = true;
                await new Promise<void>((resolve) => this.logStream.end(resolve));
            } catch (err) {
                firstError ??= err;
            }
        }

        if (firstError) {
            throw firstError;
        }
    }

    private execOrThrow = async (cmd: string[]): Promise<{ exitCode: number; output: string; stdout: string; stderr: string }> => {
        const result = await this.container.exec(cmd)
        if (result.exitCode !== 0) {
            throw new Error(
                `Command failed: ${cmd.join(' ')}\n` +
                `Exit code: ${result.exitCode}\n` +
                `Output: ${result.output}\n` +
                `Stdout: ${result.stdout}\n` +
                `Stderr: ${result.stderr}`
            )
        }
        return result
    }

    createAdmin = async (email: string, username: string, password: string) => {
        await this.execOrThrow(["mmctl", "--local", "user", "create", "--email", email, "--username", username, "--password", password, "--system-admin", "--email-verified"])
    }

    createUser = async (email: string, username: string, password: string) => {
        await this.execOrThrow(["mmctl", "--local", "user", "create", "--email", email, "--username", username, "--password", password, "--email-verified"])
    }

    createTeam = async (name: string, displayName: string) => {
        await this.execOrThrow(["mmctl", "--local", "team", "create", "--name", name, "--display-name", displayName])
    }

    addUserToTeam = async (username: string, teamname: string) => {
        await this.execOrThrow(["mmctl", "--local", "team", "users", "add", teamname, username])
    }

    getLogs = async (lines: number): Promise<string> => {
        const {output} = await this.execOrThrow(["mmctl", "--local", "logs", "--number", lines.toString()])
        return output
    }

    setSiteURL = async () => {
        const url = this.url()
        await this.execOrThrow(["mmctl", "--local", "config", "set", "ServiceSettings.SiteURL", url])
    }

    installPlugin = async (pluginPath: string, pluginID: string, pluginConfig?: PluginConfigInput) => {
        await this.container.copyFilesToContainer([{source: pluginPath, target: `/tmp/plugin.tar.gz`}])

        await this.execOrThrow(["mmctl", "--local", "plugin", "add", '/tmp/plugin.tar.gz'])
        await this.execOrThrow(["mmctl", "--local", "plugin", "enable", pluginID])

        // Set config via plugin admin API (replaces mmctl config patch)
        if (pluginConfig) {
            // Callers pass { config: {...} } — extract the inner config object
            // since the admin API expects config.Config directly
            const wrappedConfig = pluginConfig as {config?: PluginConfig};
            const configData: PluginConfig = wrappedConfig.config ?? pluginConfig;
            await this.setPluginConfig(pluginID, configData);
        }
    }

    setPluginConfig = async (pluginID: string, config: PluginConfig) => {
        const adminClient = await this.getAdminClient();
        const api = pluginAdminConfigApiFromClient(adminClient, this.url(), pluginID);
        await api.put(config as Record<string, unknown>, { settleMs: 2000 });
    }

    withEnv = (env: string, value: string): MattermostContainer => {
        this.envs[env] = value
        return this
    }

    withAdmin = (email: string, username: string, password: string): MattermostContainer => {
        this.email = email;
        this.username = username;
        this.password = password;
        return this;
    }

    withTeam = (teamName: string, teamDisplayName: string): MattermostContainer => {
        this.teamName = teamName;
        this.teamDisplayName = teamDisplayName;
        return this;
    }

    withConfigFile = (cfg: string): MattermostContainer => {
        const target = "/etc/mattermost.json";
        this.configFile = this.configFile.filter(f => f.target !== target);
        this.configFile.push({ source: cfg, target });

        // Remove any existing -c flag pair before adding the new one
        const flagIndex = this.command.indexOf("-c");
        if (flagIndex !== -1) {
            this.command.splice(flagIndex, 2);
        }
        this.command.push("-c", target);
        return this
    }

    withPlugin = (pluginPath: string, pluginID: string, pluginConfig?: PluginConfigInput): MattermostContainer => {
        this.plugins.push({id: pluginID, path: pluginPath, config: pluginConfig})
        return this
    }

    constructor() {
        this.command = ["mattermost", "server"];
        const dbconn = `postgres://user:pass@db:5432/mattermost_test?sslmode=disable`;
        this.envs = {
                "MM_SQLSETTINGS_DATASOURCE":          dbconn,
                "MM_SQLSETTINGS_DRIVERNAME":          "postgres",
                "MM_SERVICESETTINGS_ENABLELOCALMODE": "true",
                "MM_PASSWORDSETTINGS_MINIMUMLENGTH":  "5",
                "MM_PLUGINSETTINGS_ENABLEUPLOADS":    "true",
                "MM_FILESETTINGS_MAXFILESIZE":        "256000000",
                "MM_LOGSETTINGS_CONSOLELEVEL":        "DEBUG",
                "MM_LOGSETTINGS_FILELEVEL":           "DEBUG",
                "MM_SERVICESETTINGS_ENABLEDEVELOPER": "true",
                "MM_SERVICESETTINGS_ENABLETESTING":   "true",
                "MM_PLUGINSETTINGS_AUTOMATICPREPACKAGEDPLUGINS": "false",
        };
        this.plugins = [];
        this.configFile = [];
        this.email = defaultEmail;
        this.username = defaultUsername;
        this.password = defaultPassword;
        this.teamName = defaultTeamName;
        this.teamDisplayName = defaultTeamDisplayName;
        this.isLogStreamClosed = false;
    }

    start = async (): Promise<MattermostContainer> => {
        let image = defaultMattermostImage;
        const isCustomImage = !!process.env.MM_IMAGE;
        if (isCustomImage) {
            image = process.env.MM_IMAGE;
        }
        console.log(`\n🚀 Starting Mattermost container`);
        console.log(`   Image: ${image}${isCustomImage ? ' (custom via MM_IMAGE)' : ' (default)'}`);

        try {
            this.network = await new Network().start()
            this.pgContainer = await new PostgreSqlContainer("postgres:15")
                .withExposedPorts(5432)
                .withDatabase("mattermost_test")
                .withUsername("user")
                .withPassword("pass")
                .withNetworkMode(this.network.getName())
                .withWaitStrategy(Wait.forLogMessage("database system is ready to accept connections"))
                .withNetworkAliases("db")
                .start()

            this.container = await new GenericContainer(image)
                .withPlatform('linux/amd64')
                .withEnvironment(this.envs)
                .withExposedPorts(8065)
                .withNetwork(this.network)
                .withNetworkAliases("mattermost")
                .withCommand(this.command)
                .withStartupTimeout(120000)
                .withWaitStrategy(Wait.forLogMessage("Server is listening on"))
                .withCopyFilesToContainer(this.configFile)
                .withLogConsumer((stream) => {
                    // Create log file with timestamp
                    const fs = require('fs');
                    const logDir = 'logs';
                    if (!fs.existsSync(logDir)){
                        fs.mkdirSync(logDir);
                    }
                    this.logStream = fs.createWriteStream(`${logDir}/server-logs.log`, {flags: 'a'});
                    this.isLogStreamClosed = false;

                    stream.on('data', (data: string | Buffer) => {
                        const logLine = String(data);

                        // Write all logs to file
                        if (this.logStream && !this.isLogStreamClosed) {
                            this.logStream.write(logLine + '\n');
                        }

                        // Only print plugin logs to console in non-CI environments
                        if (!process.env.CI && logLine.includes('"plugin_id":"focalboard"')) {
                            console.log(logLine);
                        }
                    });
                })
                .start()

            await this.setSiteURL()
            await this.createAdmin(this.email, this.username, this.password)
            await this.createTeam(this.teamName, this.teamDisplayName)
            await this.addUserToTeam(this.username, this.teamName)

            for (const plugin of this.plugins) {
                await this.installPlugin(plugin.path, plugin.id, plugin.config)
            }

            return this
        } catch (err) {
            await this.stop().catch(() => undefined)
            throw err
        }
    }
}
