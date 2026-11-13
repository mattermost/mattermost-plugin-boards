import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import {
    createCardViaApi,
    getBlocksViaApi,
    getBoardViaApi,
    openBoard,
    seedKanbanBoard,
    STATUS_PROPERTY_ID,
    waitForBlockPatch,
    type SeededBoard,
} from 'helpers/boards';

const NOTES_PROPERTY_ID = 'notesprop';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Table View', () => {
    test.describe.configure({ timeout: 300000 });

    const { username, password } = Users.admin;
    const headerCells = (page: Page) => page.locator('.TableHeaders .header-cell');
    const rowTitles = (page: Page) =>
        page.locator('.TableRow .title-cell .Editable').evaluateAll((els) => els.map((el) => (el as HTMLInputElement).value));

    async function openTableBoard(page: Page, title: string, cardTitles: string[] = []) {
        const board = await seedKanbanBoard(
            mattermost,
            username,
            password,
            title,
            ['To Do', 'Doing'],
            [{ id: NOTES_PROPERTY_ID, name: 'Notes', type: 'text', options: [] }],
            {
                viewType: 'table',
                groupById: '',
                visiblePropertyIds: [STATUS_PROPERTY_ID, NOTES_PROPERTY_ID],
                columnWidths: { __title: 280, [STATUS_PROPERTY_ID]: 150, [NOTES_PROPERTY_ID]: 150 },
            },
        );
        for (const cardTitle of cardTitles) {
            await createCardViaApi(mattermost, board.boardId, cardTitle, board.token);
        }
        await openBoard(page, mattermost, username, password, board);
        await expect(page.locator('.Table')).toBeVisible({ timeout: 15000 });
        return board;
    }

    async function savedView(board: SeededBoard) {
        const blocks = await getBlocksViaApi(mattermost, board.boardId, board.token);
        return blocks.find((b) => b.id === board.viewId)!;
    }

    async function savedPropertyNames(board: SeededBoard) {
        return (await getBoardViaApi(mattermost, board.boardId, board.token)).cardProperties.map((p) => p.name);
    }

    async function runHeaderMenuAction(page: Page, columnName: string, action: string) {
        await headerCells(page).filter({ hasText: columnName }).locator('.Label').click();
        const menu = page.locator('.Menu.noselect');
        await expect(menu).toBeVisible({ timeout: 5000 });
        await menu.getByRole('button', { name: action, exact: true }).click();
    }

    test('can sort ascending and descending from the column header menu', async ({ page }) => {
        const board = await openTableBoard(page, 'Table Sort Board', ['Charlie', 'Alpha', 'Bravo']);
        await expect.poll(() => rowTitles(page), { timeout: 15000 }).toHaveLength(3);

        await runHeaderMenuAction(page, 'Name', 'Sort ascending');
        await expect.poll(() => rowTitles(page), { timeout: 10000 }).toEqual(['Alpha', 'Bravo', 'Charlie']);
        await expect.poll(async () => (await savedView(board)).fields.sortOptions, { timeout: 10000 })
            .toEqual([{ propertyId: '__title', reversed: false }]);

        await runHeaderMenuAction(page, 'Name', 'Sort descending');
        await expect.poll(() => rowTitles(page), { timeout: 10000 }).toEqual(['Charlie', 'Bravo', 'Alpha']);
        await expect.poll(async () => (await savedView(board)).fields.sortOptions, { timeout: 10000 })
            .toEqual([{ propertyId: '__title', reversed: true }]);
    });

    test('can insert a column to the left of a property', async ({ page }) => {
        const board = await openTableBoard(page, 'Table Insert Board');
        await expect(headerCells(page)).toHaveCount(3, { timeout: 10000 });

        await runHeaderMenuAction(page, 'Notes', 'Insert left');

        await expect(headerCells(page)).toHaveCount(4, { timeout: 10000 });
        await expect.poll(async () => (await savedPropertyNames(board)).length, { timeout: 10000 }).toBe(3);
        await expect.poll(async () => (await savedView(board)).fields.visiblePropertyIds.length, { timeout: 10000 }).toBe(3);
    });

    test('can hide a column', async ({ page }) => {
        const board = await openTableBoard(page, 'Table Hide Board');

        await runHeaderMenuAction(page, 'Notes', 'Hide');

        await expect(headerCells(page).filter({ hasText: 'Notes' })).toHaveCount(0, { timeout: 10000 });
        await expect(headerCells(page)).toHaveCount(2);
        await expect.poll(async () => (await savedView(board)).fields.visiblePropertyIds, { timeout: 10000 })
            .toEqual([STATUS_PROPERTY_ID]);
    });

    test('can duplicate a column', async ({ page }) => {
        const board = await openTableBoard(page, 'Table Duplicate Board');

        await runHeaderMenuAction(page, 'Notes', 'Duplicate');

        await expect(headerCells(page).filter({ hasText: 'Notes' })).toHaveCount(2, { timeout: 10000 });
        await expect.poll(async () => (await savedPropertyNames(board)).filter((name) => name.startsWith('Notes')).length,
            { timeout: 10000 }).toBe(2);
    });

    test('can delete a column', async ({ page }) => {
        const board = await openTableBoard(page, 'Table Delete Board');

        await runHeaderMenuAction(page, 'Notes', 'Delete');

        await expect(headerCells(page).filter({ hasText: 'Notes' })).toHaveCount(0, { timeout: 10000 });
        await expect.poll(() => savedPropertyNames(board), { timeout: 10000 }).toEqual(['Status']);
    });

    test('can add a row and edit its title inline', async ({ page }) => {
        await openTableBoard(page, 'Table Add Row Board');
        await expect(page.locator('.TableRow')).toHaveCount(0);

        await page.locator('.octo-table-footer .octo-table-cell').click();
        await expect(page.locator('.TableRow')).toHaveCount(1, { timeout: 10000 });

        // Enter on the last row would also add a new card, so save on blur instead.
        const titleInput = page.locator('.TableRow .title-cell .Editable').first();
        await titleInput.click();
        await titleInput.fill('Inline Row Title');
        const saved = waitForBlockPatch(page, (patch) => patch.title === 'Inline Row Title');
        await titleInput.blur();
        await saved;

        await page.reload({ waitUntil: 'domcontentloaded' });
        await expect.poll(() => rowTitles(page), { timeout: 15000 }).toEqual(['Inline Row Title']);
    });
});
