import { Page, Locator } from '@playwright/test';

export class MattermostPage {
    readonly page: Page;
    readonly postTextbox: Locator;
    readonly sendButton: Locator;

    constructor(page: Page) {
        this.page = page;
        this.postTextbox = page.getByTestId('post_textbox');
        this.sendButton = page.getByTestId('channel_view').getByTestId('SendMessageButton');
    }

    async login(
        url: string,
        username: string,
        password: string,
        options?: { channelViewTimeoutMs?: number },
    ) {
        const channelTimeout = options?.channelViewTimeoutMs ?? 60000;
        await this.page.addInitScript(() => { localStorage.setItem('__landingPageSeen__', 'true'); });

        // Polyfill crypto.randomUUID for insecure contexts (e.g., Docker test environments
        // where the Mattermost URL uses a non-localhost IP like http://172.17.0.1:PORT).
        await this.page.addInitScript(() => {
            if (typeof crypto !== 'undefined' && typeof crypto.randomUUID !== 'function') {
                crypto.randomUUID = function randomUUID() {
                    const bytes = new Uint8Array(16);
                    crypto.getRandomValues(bytes);
                    bytes[6] = (bytes[6] & 0x0f) | 0x40;
                    bytes[8] = (bytes[8] & 0x3f) | 0x80;
                    const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
                    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}` as `${string}-${string}-${string}-${string}-${string}`;
                };
            }
        });

        // Retry navigation with exponential backoff for flaky network conditions
        let lastError: Error | null = null;
        for (let attempt = 0; attempt < 3; attempt++) {
            try {
                await this.page.goto(url, { waitUntil: 'domcontentloaded', timeout: 60000 });
                break;
            } catch (error) {
                lastError = error as Error;
                if (attempt < 2) {
                    await this.page.waitForTimeout(1000 * (attempt + 1));
                }
            }
        }
        if (lastError && !(await this.page.getByText('Log in to your account').isVisible().catch(() => false))) {
            throw lastError;
        }

        await this.page.getByText('Log in to your account').waitFor({ timeout: 60000 });
        await this.page.getByPlaceholder('Password').fill(password);
        await this.page.getByPlaceholder("Email or Username").fill(username);
        await this.page.getByTestId('saveSetting').click();

        await this.page.waitForURL(/.*\/test\/channels\/.*/, { timeout: channelTimeout });
        await this.page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: channelTimeout });
    }

    static async loginAndWait(page: Page, url: string, username: string, password: string): Promise<MattermostPage> {
        const mmPage = new MattermostPage(page);
        await mmPage.login(url, username, password);
        return mmPage;
    }

    async navigateToBoardsFromUrl(baseUrl: string) {
        await this.page.goto(`${baseUrl}/boards`, { waitUntil: 'domcontentloaded', timeout: 60000 });

        // On first visit, boards shows a welcome page. Dismiss it if present.
        const skipLink = this.page.locator('.WelcomePage .skip');
        const boardsReady = this.page.locator('.Sidebar.octo-sidebar, .BoardTemplateSelector, .Workspace').first();
        const first = await Promise.race([
            skipLink.waitFor({ state: 'visible', timeout: 30000 }).then(() => 'welcome'),
            boardsReady.waitFor({ state: 'visible', timeout: 30000 }).then(() => 'boards'),
        ]).catch(() => 'timeout');

        if (first === 'welcome') {
            await skipLink.click();
            await boardsReady.waitFor({ state: 'visible', timeout: 30000 });
        } else if (first === 'timeout') {
            throw new Error('Boards did not load: neither the welcome page nor the boards UI appeared within 30s');
        }

        // The Boards "Welcome To Boards" tour overlay renders inside .Workspace, so
        // boardsReady may resolve while the overlay is still visible. Dismiss it if present.
        const tourSkip = this.page.locator('text=No thanks, I\'ll figure it out myself');
        if (await tourSkip.isVisible({ timeout: 2000 }).catch(() => false)) {
            await tourSkip.click();
        }
    }
}

// Legacy function for backward compatibility
export const login = async (page: Page, url: string, username: string, password: string) => {
    const mmPage = new MattermostPage(page);
    await mmPage.login(url, username, password);
};
