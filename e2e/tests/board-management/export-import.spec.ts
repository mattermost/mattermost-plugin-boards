import { readFile } from 'fs/promises';

import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import { createCardViaApi, getBlocksViaApi, listTeamBoardsViaApi, openBoard, seedBoard } from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Export and Import', () => {
    test.describe.configure({ timeout: 300000 });

    const { username, password } = Users.admin;

    test('can export a view to CSV', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'CSV Export Board');
        await createCardViaApi(mattermost, board.boardId, 'CSV Card', board.token);
        await openBoard(page, mattermost, username, password, board);

        await page.locator('.ViewHeader').getByRole('button', { name: 'View header menu' }).click();
        const downloadPromise = page.waitForEvent('download');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Export to CSV' }).click();
        const download = await downloadPromise;

        expect(download.suggestedFilename()).toMatch(/\.csv$/);
        const lines = (await readFile(await download.path(), 'utf-8')).split('\r\n');
        expect(lines[0]).toMatch(/^Name(,|$)/);
        expect(lines.some((line) => line.startsWith('"CSV Card"'))).toBe(true);
    });

    test('can export a board archive and import it as a copy of the board', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Importable Board');
        await createCardViaApi(mattermost, board.boardId, 'Imported Card', board.token);

        const boardsNamed = async () =>
            (await listTeamBoardsViaApi(mattermost, board.teamId, board.token))
                .filter((b) => b.title === 'Importable Board');
        expect(await boardsNamed()).toHaveLength(1);

        await openBoard(page, mattermost, username, password, board);

        await page.locator('.ViewHeader').getByRole('button', { name: 'View header menu' }).click();
        const downloadPromise = page.waitForEvent('download');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Export board archive' }).click();
        const download = await downloadPromise;
        expect(download.suggestedFilename()).toMatch(/\.boardarchive$/);

        await page.locator('.GlobalHeaderSettingsMenu .menu-entry').click();
        await page.locator('#import').hover();
        const chooserPromise = page.waitForEvent('filechooser');
        await page.locator('#import').getByRole('button', { name: 'Import archive', exact: true }).click();
        const chooser = await chooserPromise;
        await chooser.setFiles(await download.path());

        await expect.poll(async () => (await boardsNamed()).length, { timeout: 30000 }).toBe(2);
        const imported = (await boardsNamed()).find((b) => b.id !== board.boardId)!;
        const importedCards = (await getBlocksViaApi(mattermost, imported.id, board.token))
            .filter((b) => b.type === 'card')
            .map((b) => b.title);
        expect(importedCards).toContain('Imported Card');
    });
});
