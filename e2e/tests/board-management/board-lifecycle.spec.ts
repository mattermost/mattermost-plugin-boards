import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import {
    deleteBoardViaApi,
    openBoard,
    openSidebarBoardMenu,
    seedBoard,
    undeleteBoardViaApi,
} from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Board Lifecycle', () => {
    test.describe.configure({ timeout: 300000 });

    const { username, password } = Users.admin;
    const sidebarItems = (page: import('@playwright/test').Page) => page.locator('.SidebarBoardItem.subitem');

    test('can delete a board from the sidebar menu', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Board To Delete');
        await openBoard(page, mattermost, username, password, board);
        await expect(sidebarItems(page).filter({ hasText: 'Board To Delete' })).toBeVisible({ timeout: 10000 });

        await openSidebarBoardMenu(page, 'Board To Delete');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Delete board' }).click();

        const dialog = page.locator('.DeleteBoardDialog');
        await expect(dialog).toBeVisible({ timeout: 5000 });
        await expect(dialog.locator('h2')).toContainText('Confirm delete board');
        await dialog.getByRole('button', { name: 'Delete' }).click();

        await expect(sidebarItems(page).filter({ hasText: 'Board To Delete' })).toHaveCount(0, { timeout: 10000 });
    });

    test('cancelling the delete dialog keeps the board', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Board To Keep');
        await openBoard(page, mattermost, username, password, board);

        await openSidebarBoardMenu(page, 'Board To Keep');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Delete board' }).click();

        const dialog = page.locator('.DeleteBoardDialog');
        await expect(dialog).toBeVisible({ timeout: 5000 });
        await dialog.getByRole('button', { name: 'Cancel' }).click();

        await expect(dialog).not.toBeVisible({ timeout: 5000 });
        await expect(sidebarItems(page).filter({ hasText: 'Board To Keep' })).toBeVisible();
    });

    test('can duplicate a board', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Duplicate Source');
        await openBoard(page, mattermost, username, password, board);

        await openSidebarBoardMenu(page, 'Duplicate Source');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Duplicate board' }).click();

        await expect(sidebarItems(page).filter({ hasText: 'Duplicate Source' })).toHaveCount(2, { timeout: 15000 });
        await expect(page.locator('.BoardComponent')).toBeVisible();
    });

    test('can create a template from a board', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Template Source');
        await openBoard(page, mattermost, username, password, board);

        await openSidebarBoardMenu(page, 'Template Source');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'New template from board' }).click();

        await expect(
            page.locator('.banner').filter({ hasText: "You're editing a board template." }),
        ).toBeVisible({ timeout: 15000 });
    });

    test('a deleted board can be restored through the API', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Board To Restore');
        await deleteBoardViaApi(mattermost, board.boardId, board.token);
        await undeleteBoardViaApi(mattermost, board.boardId, board.token);

        const other = await seedBoard(mattermost, username, password, 'Restore Anchor Board');
        await openBoard(page, mattermost, username, password, other);

        await expect(sidebarItems(page).filter({ hasText: 'Board To Restore' })).toBeVisible({ timeout: 15000 });
    });
});
