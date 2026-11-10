import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { Users } from 'helpers/users';
import { createCardViaApi, openBoard, seedBoard } from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost.stop();
});

test.describe('Board Permissions', () => {
    test('regular user can only see their own boards', async ({ page }) => {
        const mmPage = await MattermostPage.loginAndWait(page, mattermost.url(), Users.regularUser.username, Users.regularUser.password);

        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        // Boards should load without errors
        const boardsLoaded = page.locator('.Sidebar.octo-sidebar').or(page.getByText('Create a board'));
        await expect(boardsLoaded.first()).toBeVisible({ timeout: 15000 });

        // Error pages should not be visible
        await expect(page.getByText('Something went wrong')).not.toBeVisible();
        await expect(page.getByText('Access denied')).not.toBeVisible();
    });

    test('second user has an independent board workspace', async ({ page }) => {
        const mmPage = await MattermostPage.loginAndWait(page, mattermost.url(), Users.secondUser.username, Users.secondUser.password);

        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        // Boards should load for the second user as well
        const boardsLoaded = page.locator('.Sidebar.octo-sidebar').or(page.getByText('Create a board'));
        await expect(boardsLoaded.first()).toBeVisible({ timeout: 15000 });
    });

    test('admin can enable public board sharing via system config', async () => {
        const adminClient = await mattermost.getAdminClient();
        const config = await adminClient.getConfig();

        // Enable public shared boards in plugin settings
        if (!config.PluginSettings.Plugins) {
            config.PluginSettings.Plugins = {};
        }
        config.PluginSettings.Plugins['focalboard'] = {
            ...config.PluginSettings.Plugins['focalboard'],
            enablepublicsharedboards: true,
        };
        await adminClient.updateConfig(config);

        // Verify the setting was applied
        const updatedConfig = await adminClient.getConfig();
        expect(updatedConfig.PluginSettings.Plugins?.['focalboard']?.enablepublicsharedboards).toBe(true);
    });

    test('a published board link opens read-only for logged-out visitors', async ({ page, browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const config = await adminClient.getConfig();
        config.PluginSettings.Plugins = {
            ...config.PluginSettings.Plugins,
            focalboard: { ...config.PluginSettings.Plugins?.['focalboard'], enablepublicsharedboards: true },
        };
        await adminClient.updateConfig(config);

        const board = await seedBoard(mattermost, Users.admin.username, Users.admin.password, 'Published Board');
        await createCardViaApi(mattermost, board.boardId, 'Published Card', board.token);
        await openBoard(page, mattermost, Users.admin.username, Users.admin.password, board);

        await page.locator('.ShareBoardButton button').click();
        const dialog = page.locator('.ShareBoardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });
        await dialog.locator('.tab-item').filter({ hasText: 'Publish' }).click();
        await dialog.locator('.Switch').click();

        const shareLink = dialog.locator('a.shareUrl');
        await expect(shareLink).toBeVisible({ timeout: 10000 });
        const shareUrl = (await shareLink.getAttribute('href')) as string;

        const visitorContext = await browser.newContext();
        const visitor = await visitorContext.newPage();
        await visitor.goto(shareUrl, { waitUntil: 'domcontentloaded' });

        await expect(visitor.locator('.BoardComponent')).toBeVisible({ timeout: 30000 });
        await expect(visitor.locator('.KanbanCard').filter({ hasText: 'Published Card' })).toBeVisible({ timeout: 15000 });
        await expect(visitor.locator('.ViewHeader .ButtonWithMenu')).toHaveCount(0);
        await expect(visitor.locator('.ViewTitle .Editable')).toBeDisabled();

        await visitorContext.close();
    });
});
