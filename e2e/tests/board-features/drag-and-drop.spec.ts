import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import { createCardViaApi, dragAndDrop, openBoard, seedBoard, seedKanbanBoard, STATUS_PROPERTY_ID } from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Drag and Drop', () => {
    test.describe.configure({ timeout: 300000 });

    const { username, password } = Users.admin;

    const columnHeaderValues = (page: Page) =>
        page.locator('.KanbanColumnHeader').evaluateAll((els) =>
            els.map((el) => (el.querySelector('input.Editable') as HTMLInputElement | null)?.value ?? ''));

    async function columnIndex(page: Page, name: string): Promise<number> {
        await expect.poll(async () => (await columnHeaderValues(page)).indexOf(name), { timeout: 15000 })
            .toBeGreaterThanOrEqual(0);
        return (await columnHeaderValues(page)).indexOf(name);
    }

    const cardTitles = (page: Page, columnIdx: number) =>
        page.locator('.octo-board-body .octo-board-column').nth(columnIdx).locator('.KanbanCard')
            .evaluateAll((els) => els.map((el) => (el.textContent || '').trim()));

    test('dragging a card to another column changes its status', async ({ page }) => {
        const board = await seedKanbanBoard(mattermost, username, password, 'DnD Column Board');
        await createCardViaApi(mattermost, board.boardId, 'Drag Me', board.token, {
            [STATUS_PROPERTY_ID]: board.optionIds['To Do'],
        });
        await openBoard(page, mattermost, username, password, board);

        const fromIdx = await columnIndex(page, 'To Do');
        const toIdx = await columnIndex(page, 'Doing');
        await expect.poll(() => cardTitles(page, fromIdx), { timeout: 15000 }).toEqual(['Drag Me']);

        const card = page.locator('.KanbanCard').filter({ hasText: 'Drag Me' });
        const targetColumn = page.locator('.octo-board-body .octo-board-column').nth(toIdx);
        await dragAndDrop(page, card, targetColumn);

        await expect.poll(() => cardTitles(page, toIdx), { timeout: 10000 }).toEqual(['Drag Me']);
        expect(await cardTitles(page, fromIdx)).toEqual([]);

        await page.reload({ waitUntil: 'domcontentloaded' });
        const reloadedToIdx = await columnIndex(page, 'Doing');
        await expect.poll(() => cardTitles(page, reloadedToIdx), { timeout: 15000 }).toEqual(['Drag Me']);
    });

    test('dragging a card onto another card reorders them within a column', async ({ page }) => {
        const board = await seedKanbanBoard(mattermost, username, password, 'DnD Reorder Board');
        const todo = { [STATUS_PROPERTY_ID]: board.optionIds['To Do'] };
        await createCardViaApi(mattermost, board.boardId, 'Reorder One', board.token, todo);
        await createCardViaApi(mattermost, board.boardId, 'Reorder Two', board.token, todo);
        await openBoard(page, mattermost, username, password, board);

        const idx = await columnIndex(page, 'To Do');
        await expect.poll(async () => (await cardTitles(page, idx)).length, { timeout: 15000 }).toBe(2);
        const before = await cardTitles(page, idx);

        const column = page.locator('.octo-board-body .octo-board-column').nth(idx);
        await dragAndDrop(page, column.locator('.KanbanCard').last(), column.locator('.KanbanCard').first());

        await expect.poll(() => cardTitles(page, idx), { timeout: 10000 }).toEqual([...before].reverse());

        await page.reload({ waitUntil: 'domcontentloaded' });
        const reloadedIdx = await columnIndex(page, 'To Do');
        await expect.poll(() => cardTitles(page, reloadedIdx), { timeout: 15000 }).toEqual([...before].reverse());
    });

    test('boards can be reordered in the sidebar with the keyboard', async ({ page }) => {
        const first = await seedBoard(mattermost, username, password, 'Sidebar Order A');
        await seedBoard(mattermost, username, password, 'Sidebar Order B');
        await openBoard(page, mattermost, username, password, first);

        const items = page.locator('.SidebarBoardItem.subitem');
        const titles = () => items.locator('.octo-sidebar-title').allTextContents();
        await expect.poll(async () => (await titles()).length, { timeout: 15000 }).toBeGreaterThanOrEqual(2);
        const before = await titles();
        const movedTitle = before[before.length - 1];

        const lastItem = items.last();
        await lastItem.focus();
        await page.keyboard.press('Space');
        await page.keyboard.press('ArrowUp');
        await page.keyboard.press('Space');

        await expect.poll(async () => (await titles()).indexOf(movedTitle), { timeout: 10000 })
            .toBe(before.length - 2);

        await page.reload({ waitUntil: 'domcontentloaded' });
        await expect.poll(async () => (await titles()).length, { timeout: 15000 }).toBe(before.length);
        await expect.poll(async () => (await titles()).indexOf(movedTitle), { timeout: 10000 })
            .toBe(before.length - 2);
    });
});
