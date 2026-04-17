import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { createBoardViaApi } from 'helpers/boards';

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

    async function setupBoard(page: Page, username: string, password: string, boardTitle: string): Promise<void> {
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), username, password);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        const boardComponent = page.locator('.BoardComponent');
        const sidebarItem = page.locator('.octo-sidebar-list .octo-sidebar-item').filter({ hasText: boardTitle });

        // If the board already exists in the sidebar, click it directly.
        if (await sidebarItem.isVisible({ timeout: 3000 }).catch(() => false)) {
            await sidebarItem.click();
        } else {
            // Board not found — create it via the template selector.
            await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
            await page.locator('.templates-sidebar__footer button').click();
            await expect(boardComponent).toBeVisible({ timeout: 15000 });

            await expect(page.locator('.ViewTitle .Editable.title')).toHaveAttribute('placeholder', /[Uu]ntitled/, { timeout: 10000 });
            const titleInput = page.locator('.ViewTitle .Editable');
            await titleInput.click();
            await titleInput.fill(boardTitle);
            await titleInput.press('Enter');

            await expect(page.locator('.octo-sidebar-list')).toContainText(boardTitle, { timeout: 10000 });
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
        await setupBoard(page, username, password, 'Test Board');

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
        await setupBoard(page, username, password, 'Test Board');

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

        // Create first board and name it
        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
        await page.locator('.templates-sidebar__footer button').click();
        await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 15000 });

        const titleInput = page.locator('.ViewTitle .Editable');
        await titleInput.click();
        await titleInput.fill('First Board');
        await titleInput.press('Enter');
        await expect(page.locator('.octo-sidebar-list')).toContainText('First Board', { timeout: 10000 });

        // Create second board via API (the Add Board Dropdown is unreliable due to React event timing)
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        await createBoardViaApi(mattermost, 'Second Board', token);

        // WebSocket push updates the sidebar automatically; both boards should be listed
        const sidebarList = page.locator('.octo-sidebar-list');
        await expect(sidebarList).toContainText('First Board', { timeout: 15000 });
        await expect(sidebarList).toContainText('Second Board', { timeout: 15000 });
    });
});
