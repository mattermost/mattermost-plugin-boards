import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import { createCardViaApi, listTeamBoardsViaApi, openBoard, seedBoard } from 'helpers/boards';

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
    });

    test('can export a board archive', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Archive Export Board');
        await openBoard(page, mattermost, username, password, board);

        await page.locator('.ViewHeader').getByRole('button', { name: 'View header menu' }).click();
        const downloadPromise = page.waitForEvent('download');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Export board archive' }).click();
        const download = await downloadPromise;

        expect(download.suggestedFilename()).toMatch(/\.boardarchive$/);
    });

    test('can import a board archive and get a copy of the board', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Importable Board');
        await createCardViaApi(mattermost, board.boardId, 'Imported Card', board.token);

        const exportResp = await fetch(
            `${mattermost.url()}/plugins/focalboard/api/v2/boards/${board.boardId}/archive/export`,
            { headers: { Authorization: `Bearer ${board.token}`, 'X-Requested-With': 'XMLHttpRequest' } },
        );
        expect(exportResp.ok).toBe(true);
        const archive = Buffer.from(await exportResp.arrayBuffer());

        const countBoards = async () =>
            (await listTeamBoardsViaApi(mattermost, board.teamId, board.token))
                .filter((b) => b.title === 'Importable Board').length;
        expect(await countBoards()).toBe(1);

        await openBoard(page, mattermost, username, password, board);

        await page.locator('.GlobalHeaderSettingsMenu .menu-entry').click();
        await page.locator('#import').hover();
        const chooserPromise = page.waitForEvent('filechooser');
        await page.locator('#import').getByRole('button', { name: 'Import archive', exact: true }).click();
        const chooser = await chooserPromise;
        await chooser.setFiles({
            name: 'e2e.boardarchive',
            mimeType: 'application/octet-stream',
            buffer: archive,
        });

        await expect.poll(countBoards, { timeout: 30000 }).toBe(2);
    });
});
