import { test, expect } from '@playwright/test';

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
    await mattermost.stop();
});

test.describe('Board Creation', () => {
    test.describe.configure({ timeout: 300000 });
    test('boards product is accessible after login', async ({ page }) => {
        const mmPage = await MattermostPage.loginAndWait(page, mattermost.url(), username, password);

        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        // The template selector is the empty state when no boards exist
        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
    });

    test('can create an empty board', async ({ page }) => {
        const mmPage = await MattermostPage.loginAndWait(page, mattermost.url(), username, password);

        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        // When no boards exist the template selector is already shown as the empty state.
        // The "Create empty board" button lives in the sidebar footer of the template selector.
        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
        await page.locator('.templates-sidebar__footer button').click();

        // After creation the board editor should be visible
        await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 15000 });
    });

    test('new board appears in the sidebar', async ({ page }) => {
        const mmPage = await MattermostPage.loginAndWait(page, mattermost.url(), 'admin', 'admin');

        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
        await page.locator('.templates-sidebar__footer button').click();

        // After creation the sidebar should list the new board
        await expect(page.locator('.octo-sidebar-list .octo-sidebar-item').first()).toBeVisible({ timeout: 15000 });
    });
});
