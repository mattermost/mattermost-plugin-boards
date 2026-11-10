import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import { createCardViaApi, openBoard, seedKanbanBoard, STATUS_PROPERTY_ID } from 'helpers/boards';

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

    async function openHeaderMenu(page: Page, columnName: string) {
        await headerCells(page).filter({ hasText: columnName }).locator('.Label').click();
        await expect(page.locator('.Menu.noselect')).toBeVisible({ timeout: 5000 });
    }

    test('can sort ascending and descending from the column header menu', async ({ page }) => {
        await openTableBoard(page, 'Table Sort Board', ['Charlie', 'Alpha', 'Bravo']);
        await expect.poll(() => rowTitles(page), { timeout: 15000 }).toHaveLength(3);

        await openHeaderMenu(page, 'Name');
        await page.locator('#sortAscending').click();
        await expect.poll(() => rowTitles(page), { timeout: 10000 }).toEqual(['Alpha', 'Bravo', 'Charlie']);

        await openHeaderMenu(page, 'Name');
        await page.locator('#sortDescending').click();
        await expect.poll(() => rowTitles(page), { timeout: 10000 }).toEqual(['Charlie', 'Bravo', 'Alpha']);
    });

    test('can insert a column to the left of a property', async ({ page }) => {
        await openTableBoard(page, 'Table Insert Board');
        await expect(headerCells(page)).toHaveCount(3, { timeout: 10000 });

        await openHeaderMenu(page, 'Notes');
        await page.locator('#insertLeft').click();

        await expect(headerCells(page)).toHaveCount(4, { timeout: 10000 });
    });

    test('can hide a column', async ({ page }) => {
        await openTableBoard(page, 'Table Hide Board');

        await openHeaderMenu(page, 'Notes');
        await page.locator('.Menu.noselect #hide').click();

        await expect(headerCells(page).filter({ hasText: 'Notes' })).toHaveCount(0, { timeout: 10000 });
        await expect(headerCells(page)).toHaveCount(2);
    });

    test('can duplicate a column', async ({ page }) => {
        await openTableBoard(page, 'Table Duplicate Board');

        await openHeaderMenu(page, 'Notes');
        await page.locator('.Menu.noselect #duplicate').click();

        await expect(headerCells(page).filter({ hasText: 'Notes' })).toHaveCount(2, { timeout: 10000 });
    });

    test('can delete a column', async ({ page }) => {
        await openTableBoard(page, 'Table Delete Board');

        await openHeaderMenu(page, 'Notes');
        await page.locator('.Menu.noselect #delete').click();

        await expect(headerCells(page).filter({ hasText: 'Notes' })).toHaveCount(0, { timeout: 10000 });
    });

    test('can add a row and edit its title inline', async ({ page }) => {
        await openTableBoard(page, 'Table Add Row Board');
        await expect(page.locator('.TableRow')).toHaveCount(0);

        await page.locator('.octo-table-footer .octo-table-cell').click();
        await expect(page.locator('.TableRow')).toHaveCount(1, { timeout: 10000 });

        const titleInput = page.locator('.TableRow .title-cell .Editable').first();
        await titleInput.click();
        await titleInput.fill('Inline Row Title');
        await titleInput.press('Enter');

        await page.reload({ waitUntil: 'domcontentloaded' });
        await expect.poll(() => rowTitles(page), { timeout: 15000 }).toEqual(['Inline Row Title']);
    });
});
