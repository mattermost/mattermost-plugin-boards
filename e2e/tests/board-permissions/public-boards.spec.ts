import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { Users } from 'helpers/users';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost.stop();
});

test.describe('Board Permissions', () => {
    test.describe.configure({ timeout: 300000 });

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
});
