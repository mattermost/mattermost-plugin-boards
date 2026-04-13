import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';

const username = 'regularuser';
const password = 'regularuser';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Board Management', () => {
    test.describe.configure({ timeout: 300000 });

    async function createBoard(page: Page, username: string, password: string): Promise<void> {
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), username, password);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        const boardComponent = page.locator('.BoardComponent');
        const boardAlreadyLoaded = await boardComponent
            .waitFor({ state: 'visible', timeout: 5000 })
            .then(() => true)
            .catch(() => false);

        if (!boardAlreadyLoaded) {
            await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
            await page.locator('.templates-sidebar__footer button').click();
            await expect(boardComponent).toBeVisible({ timeout: 15000 });

            // New board: verify untitled placeholder shows on the right, then name it.
            await expect(page.locator('.ViewTitle .Editable.title')).toHaveAttribute('placeholder', /[Uu]ntitled/, { timeout: 10000 });
            const titleInput = page.locator('.ViewTitle .Editable');
            await titleInput.click();
            await titleInput.fill('Test Board');
            await titleInput.press('Enter');

            // Verify the name now appears in the sidebar.
            await expect(page.locator('.octo-sidebar-list')).toContainText('Test Board', { timeout: 10000 });
        }

        await expect(boardComponent).toBeVisible({ timeout: 15000 });
    }

    test('new board has default title placeholder', async ({ page }) => {
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), username, password);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
        await page.locator('.templates-sidebar__footer button').click();
        await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 15000 });

        // The board title input should show the "Untitled board" placeholder when empty
        await expect(page.locator('.ViewTitle .Editable.title')).toHaveAttribute('placeholder', /[Uu]ntitled/, { timeout: 10000 });
    });

    test('can rename a board', async ({ page }) => {
        await createBoard(page, username, password);

        // Click the board title to make it editable
        const titleLocator = page.locator('.ViewTitle .Editable');
        await titleLocator.click();
        await titleLocator.fill('My Renamed Board');

        // Commit the rename by pressing Enter
        await titleLocator.press('Enter');

        // The sidebar should now reflect the new board name
        await expect(page.locator('.octo-sidebar-list')).toContainText('My Renamed Board', { timeout: 10000 });
    });

    test('renamed board title persists after navigation', async ({ page }) => {
        await createBoard(page, username, password);

        // Rename the board
        const titleLocator = page.locator('.ViewTitle .Editable');
        await titleLocator.click();
        await titleLocator.fill('Persistent Board');
        await titleLocator.press('Enter');

        await expect(page.locator('.octo-sidebar-list')).toContainText('Persistent Board', { timeout: 10000 });

        // Navigate away by going to the main Mattermost channel, then back
        await page.goto(mattermost.url());
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await new MattermostPage(page).navigateToBoardsFromUrl(mattermost.url());

        // The renamed board should still appear in the sidebar
        await expect(page.locator('.octo-sidebar-list')).toContainText('Persistent Board', { timeout: 15000 });
    });

    test('multiple boards appear in sidebar', async ({ page }) => {
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), 'admin', 'admin');
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        // Create first board
        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
        await page.locator('.templates-sidebar__footer button').click();
        await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 15000 });

        // Create second board via REST API (the Add Board Dropdown is unreliable due to React event timing)
        const adminClient = await mattermost.getAdminClient();
        const teams = await adminClient.getMyTeams();
        const teamId = teams[0]?.id;
        expect(teamId).toBeTruthy();

        const resp = await page.request.post(
            `${mattermost.url()}/plugins/focalboard/api/v2/boards`,
            {
                headers: {
                    'Authorization': `Bearer ${(adminClient as any).token}`,
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest',
                },
                data: { teamId, title: 'Second Board', type: 'O' },
            }
        );
        expect(resp.ok()).toBeTruthy();

        // WebSocket push updates the sidebar automatically
        await expect(page.locator('.octo-sidebar-list')).toContainText('Second Board', { timeout: 15000 });
    });
});
