import { test, expect, type Locator, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import { openBoard, seedBoard } from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Board Search and Switcher', () => {
    test.describe.configure({ timeout: 300000 });

    const { username, password } = Users.admin;

    /** Type a query and return the board titles from a successful search response. */
    async function search(page: Page, dialog: Locator, query: string): Promise<string[]> {
        const responsePromise = page.waitForResponse((response) => {
            const url = new URL(response.url());
            return url.pathname.endsWith('/boards/search') && url.searchParams.get('q') === query;
        });
        await dialog.locator('input.searchQuery').fill(query);
        const response = await responsePromise;
        expect(response.status()).toBe(200);
        const boards = await response.json();
        expect(Array.isArray(boards)).toBe(true);
        return (boards as Array<{ title: string }>).map((b) => b.title);
    }

    test('the Find Boards dialog finds a board by title and navigates to it', async ({ page }) => {
        const anchor = await seedBoard(mattermost, username, password, 'Switcher Anchor Board');
        const target = await seedBoard(mattermost, username, password, 'Switcher Target Board');
        await openBoard(page, mattermost, username, password, anchor);

        await page.locator('.BoardsSwitcher').click();
        const dialog = page.locator('.BoardSwitcherDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        await dialog.locator('input.searchQuery').fill('Switcher Target');
        const result = dialog.locator('.blockSearchResult').filter({ hasText: 'Switcher Target Board' });
        await expect(result).toBeVisible({ timeout: 10000 });
        await expect(dialog.locator('.blockSearchResult').filter({ hasText: 'Switcher Anchor Board' })).toHaveCount(0);

        await result.click();

        await expect(dialog).not.toBeVisible({ timeout: 5000 });
        await expect(page).toHaveURL(new RegExp(`/${target.boardId}`), { timeout: 10000 });
        await expect(page.locator('.ViewTitle .Editable')).toHaveValue('Switcher Target Board', { timeout: 15000 });
    });

    test('the Find Boards dialog opens with the keyboard shortcut and closes with Escape', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Shortcut Board');
        await openBoard(page, mattermost, username, password, board);

        await page.keyboard.press('ControlOrMeta+k');
        const dialog = page.locator('.BoardSwitcherDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        await page.keyboard.press('Escape');
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    test('the Find Boards dialog shows an empty state when nothing matches', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'No Match Anchor');
        await openBoard(page, mattermost, username, password, board);

        await page.locator('.BoardsSwitcher').click();
        const dialog = page.locator('.BoardSwitcherDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        expect(await search(page, dialog, 'zzz-no-such-board-zzz')).toEqual([]);
        await expect(dialog.locator('.noResults:not(.introScreen)')).toBeVisible({ timeout: 10000 });
        await expect(dialog.locator('.blockSearchResult')).toHaveCount(0);
    });

    test('a private board is not found by a non-member', async ({ page }) => {
        await seedBoard(mattermost, username, password, 'Hidden Private Board', 'P');
        await seedBoard(mattermost, username, password, 'Hidden Open Board');
        const visible = await seedBoard(mattermost, Users.regularUser.username, Users.regularUser.password, 'Visible Own Board');
        await openBoard(page, mattermost, Users.regularUser.username, Users.regularUser.password, visible);

        await page.locator('.BoardsSwitcher').click();
        const dialog = page.locator('.BoardSwitcherDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        const titles = await search(page, dialog, 'Hidden');
        expect(titles).toContain('Hidden Open Board');
        expect(titles).not.toContain('Hidden Private Board');
        await expect(dialog.locator('.blockSearchResult').filter({ hasText: 'Hidden Open Board' })).toBeVisible({ timeout: 10000 });
        await expect(dialog.locator('.blockSearchResult').filter({ hasText: 'Hidden Private Board' })).toHaveCount(0);
    });
});
