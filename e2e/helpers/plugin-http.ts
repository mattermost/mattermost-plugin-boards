/**
 * HTTP helpers for Mattermost plugin APIs (admin config and plugin routes).
 */

import { Client4 } from '@mattermost/client';

const DEFAULT_PUT_SETTLE_MS = 500;

function stripTrailingSlash(url: string): string {
    return url.replace(/\/$/, '');
}

async function readErrorBody(response: Response): Promise<string> {
    try {
        return await response.text();
    } catch {
        return response.statusText;
    }
}

/**
 * GET/PUT /plugins/{pluginId}/admin/config — database-backed plugin configuration.
 */
export class PluginAdminConfigApi {
    constructor(
        private readonly baseUrl: string,
        private readonly getToken: () => string,
        private readonly pluginId: string,
    ) {}

    private adminConfigUrl(): string {
        return `${stripTrailingSlash(this.baseUrl)}/plugins/${this.pluginId}/admin/config`;
    }

    async get(): Promise<Record<string, unknown>> {
        const response = await fetch(this.adminConfigUrl(), {
            method: 'GET',
            headers: {
                Authorization: `Bearer ${this.getToken()}`,
            },
        });

        if (!response.ok) {
            const text = await readErrorBody(response);
            throw new Error(`Plugin ${this.pluginId} configuration not found: ${response.status} ${text}`);
        }

        return (await response.json()) as Record<string, unknown>;
    }

    /**
     * @param options.settleMs - Wait after success so listeners can apply.
     */
    async put(config: Record<string, unknown>, options?: { settleMs?: number }): Promise<void> {
        const response = await fetch(this.adminConfigUrl(), {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${this.getToken()}`,
            },
            body: JSON.stringify(config),
        });

        if (!response.ok) {
            const text = await readErrorBody(response);
            throw new Error(`Failed to update plugin config: ${response.status} ${text}`);
        }

        const settleMs = options?.settleMs ?? DEFAULT_PUT_SETTLE_MS;
        await new Promise((resolve) => setTimeout(resolve, settleMs));
    }
}

export function pluginAdminConfigApiFromClient(
    client: Client4,
    baseUrl: string,
    pluginId: string,
): PluginAdminConfigApi {
    return new PluginAdminConfigApi(baseUrl, () => client.getToken(), pluginId);
}
