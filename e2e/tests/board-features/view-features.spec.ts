import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { Users } from 'helpers/users';
import { createCardViaApi, openBoard, seedKanbanBoard, STATUS_PROPERTY_ID } from 'helpers/boards';

const username = 'regularuser';
const password = 'regularuser';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Board View Features', () => {
    test.describe.configure({ timeout: 300000 });

    /**
     * Log in, navigate to boards, and ensure a board is loaded.
     * Creates one if the user has none yet (empty state).
     */
    async function createBoard(page: Page): Promise<void> {
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), username, password);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        const boardComponent = page.locator('.BoardComponent');
        const boardAlreadyLoaded = await boardComponent
            .waitFor({ state: 'visible', timeout: 5000 })
            .then(() => true)
            .catch(() => false);

        if (!boardAlreadyLoaded) {
            await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
            await page.locator('.templates-sidebar__footer button').click();
            await expect(boardComponent).toBeVisible({ timeout: 15000 });

            await expect(page.locator('.ViewTitle .Editable.title')).toHaveAttribute('placeholder', /[Uu]ntitled/, { timeout: 10000 });
            const titleInput = page.locator('.ViewTitle .Editable');
            await titleInput.click();
            await titleInput.fill('Test Board');
            await titleInput.press('Enter');

            await expect(page.locator('.octo-sidebar-list')).toContainText('Test Board', { timeout: 10000 });
        }

        await expect(boardComponent).toBeVisible({ timeout: 15000 });
    }

    /**
     * Create a new card and give it a name.
     * The view-header New control is a div.ButtonWithMenu, not a <button>.
     * getByRole('button', { name: 'New' }) matches every kanban column's "+ New".
     */
    async function createNamedCard(page: Page, cardName: string): Promise<void> {
        await page.locator('.ViewHeader .ButtonWithMenu .button-text').click();

        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        const titleField = dialog.locator('.CardDetail .Editable.title:not([disabled])');
        await titleField.click();
        await titleField.fill(cardName);

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });

        await expect(page.locator('.KanbanCard').filter({ hasText: cardName })).toBeVisible({ timeout: 10000 });
    }

    /**
     * Create a new card and give it a brand-new Status option, which also
     * registers that option on the board so it becomes selectable in filters.
     */
    async function createCardWithStatus(page: Page, cardName: string, statusName: string): Promise<void> {
        await page.locator('.ViewHeader .ButtonWithMenu .button-text').click();

        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        const titleField = dialog.locator('.CardDetail .Editable.title:not([disabled])');
        await titleField.click();
        await titleField.fill(cardName);

        // Open the Status value selector and create a new option by name.
        await dialog.locator('.octo-propertyvalue').first().click();
        const valueInput = page.locator('.ValueSelector input').last();
        await valueInput.fill(statusName);
        await valueInput.press('Enter');

        // Confirm the option was actually created and selected before closing, so
        // a setup failure surfaces here rather than later in the filter menu.
        await expect(dialog.locator('.octo-propertyvalue').first()).toContainText(statusName, { timeout: 5000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Sort
    // ─────────────────────────────────────────────────────────────────────────

    test('can sort cards by name ascending', async ({ page }) => {
        await createBoard(page);

        // Create Apple first (older), then Zebra (newer).
        // Focalboard places the newest card at the top of a column,
        // so without a sort the order is Zebra → Apple.
        // After adding a Name ↑ sort, Apple (A < Z) must come first.
        await createNamedCard(page, 'Apple Card');
        await createNamedCard(page, 'Zebra Card');

        // Open the Sort menu and pick "Name" (the implicit title property).
        await page.locator('.ViewHeader').getByRole('button', { name: 'Sort' }).click();
        const sortMenu = page.locator('.Menu.noselect');
        await expect(sortMenu).toBeVisible({ timeout: 5000 });
        await sortMenu.getByRole('button', { name: 'Name' }).click();

        // Apple should now appear before Zebra in the DOM.
        await expect(page.locator('.KanbanCard').first()).toContainText('Apple Card', { timeout: 10000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Search box
    // ─────────────────────────────────────────────────────────────────────────

    test('search box filters visible cards', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'SearchTarget Card');
        await createNamedCard(page, 'OtherItem Card');

        // Type in the inline search box to narrow visible cards.
        const searchBox = page.locator('input[placeholder="Search cards"]');
        await searchBox.fill('SearchTarget');

        await expect(page.locator('.KanbanCard').filter({ hasText: 'SearchTarget Card' })).toBeVisible({ timeout: 5000 });
        await expect(page.locator('.KanbanCard').filter({ hasText: 'OtherItem Card' })).not.toBeVisible({ timeout: 5000 });

        // Clearing restores all cards.
        await searchBox.clear();
        await expect(page.locator('.KanbanCard').filter({ hasText: 'OtherItem Card' })).toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Filter button
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a filter and remove it', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'FilterTest Card');

        // Open the Filter panel.
        await page.locator('.ViewHeader').getByRole('button', { name: 'Filter' }).click();

        // Add a filter entry (defaults to: Status includes …).
        await page.getByRole('button', { name: '+ Add filter' }).click();

        // Change the condition from "includes" to "is not empty".
        // Every card has an empty Status, so this hides all of them.
        const conditionBtn = page.locator('.FilterEntry').last().getByRole('button', { name: 'includes' });
        await conditionBtn.click();
        await page.locator('.Menu.noselect').getByRole('button', { name: 'is not empty' }).click();

        // All cards should be hidden (Status is empty on every card).
        await expect(page.locator('.KanbanCard').filter({ hasText: 'FilterTest Card' })).not.toBeVisible({ timeout: 10000 });

        // Remove the filter via the Delete button inside the FilterEntry row.
        await page.locator('.FilterEntry').last().getByRole('button', { name: 'Delete' }).click();

        // Close the filter panel.
        await page.keyboard.press('Escape');

        // Cards should be visible again.
        await expect(page.locator('.KanbanCard').filter({ hasText: 'FilterTest Card' })).toBeVisible({ timeout: 10000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Properties panel
    // ─────────────────────────────────────────────────────────────────────────

    test('properties panel opens and lists board properties', async ({ page }) => {
        await createBoard(page);

        // Open the Properties menu from the view toolbar.
        await page.locator('.ViewHeader').getByRole('button', { name: 'Properties', exact: true }).click();

        // The panel should appear and list the default Status property.
        const propertiesMenu = page.locator('.Menu.noselect');
        await expect(propertiesMenu).toBeVisible({ timeout: 5000 });
        await expect(propertiesMenu).toContainText('Status', { timeout: 5000 });

        // Close the menu by clicking the Properties button again (toggle).
        await page.locator('.ViewHeader').getByRole('button', { name: 'Properties', exact: true }).click();
        await expect(propertiesMenu).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Card face properties
    // ─────────────────────────────────────────────────────────────────────────

    test('property value is visible on kanban card face after closing the dialog', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'PropDisplay Card');

        // Open the card and set a Status value by creating a new option.
        await page.locator('.KanbanCard').filter({ hasText: 'PropDisplay Card' }).click();
        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        // Click the Status property value cell to open the ValueSelector.
        await dialog.locator('.octo-propertyvalue').first().click();

        // Type a new option name and press Enter to create and select it.
        const valueInput = page.locator('.ValueSelector input').last();
        await valueInput.fill('In Progress');
        await valueInput.press('Enter');

        // Close the card dialog.
        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });

        // Enable the Status property in the Properties menu so it shows on card faces.
        await page.locator('.ViewHeader').getByRole('button', { name: 'Properties', exact: true }).click();
        const propertiesMenu = page.locator('.Menu.noselect');
        await expect(propertiesMenu).toBeVisible({ timeout: 5000 });

        // Turn Status on if it is currently off.
        const statusItem = propertiesMenu.locator('.SwitchOption').filter({ hasText: 'Status' });
        const statusSwitch = statusItem.locator('.Switch');
        const isAlreadyOn = await statusSwitch.evaluate((el) => el.classList.contains('on'));
        if (!isAlreadyOn) {
            await statusSwitch.click();
        }

        // Close the Properties menu.
        await page.locator('.ViewHeader').getByRole('button', { name: 'Properties', exact: true }).click();
        await expect(propertiesMenu).not.toBeVisible({ timeout: 5000 });

        // The "In Progress" label must appear on the kanban card face without reopening the dialog.
        await expect(
            page.locator('.KanbanCard').filter({ hasText: 'PropDisplay Card' }).locator('.Label'),
        ).toContainText('In Progress', { timeout: 10000 });
    });

    test.describe('kanban columns and grouping', () => {
        const columnHeaderValues = (page: Page) =>
            page.locator('.KanbanColumnHeader').evaluateAll((els) =>
                els.map((el) => (el.querySelector('input.Editable') as HTMLInputElement | null)?.value ?? ''));

        async function columnHeader(page: Page, name: string) {
            await expect.poll(async () => (await columnHeaderValues(page)).indexOf(name), { timeout: 15000 })
                .toBeGreaterThanOrEqual(0);
            const index = (await columnHeaderValues(page)).indexOf(name);
            return page.locator('.KanbanColumnHeader').nth(index);
        }

        async function openKanbanBoard(
            page: Page,
            title: string,
            extraProperties: Array<Record<string, unknown>> = [],
        ) {
            const board = await seedKanbanBoard(
                mattermost,
                Users.admin.username,
                Users.admin.password,
                title,
                ['To Do', 'Doing', 'Done'],
                extraProperties,
            );
            await openBoard(page, mattermost, Users.admin.username, Users.admin.password, board);
            return board;
        }

        test('can change the group-by property', async ({ page }) => {
            await openKanbanBoard(page, 'Group By Board', [{
                id: 'prioprop',
                name: 'Priority',
                type: 'select',
                options: [{ id: 'prioHigh', value: 'High', color: 'propColorDefault' }],
            }]);
            await columnHeader(page, 'Doing');

            await page.locator('.ViewHeader').getByRole('button', { name: /Group by/ }).click();
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Priority' }).click();

            await columnHeader(page, 'High');
            expect(await columnHeaderValues(page)).not.toContain('Doing');
        });

        test('can hide a kanban column and show it again', async ({ page }) => {
            await openKanbanBoard(page, 'Hide Column Board');

            const header = await columnHeader(page, 'Doing');
            await header.locator('.MenuWrapper button.IconButton').first().click();
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Hide' }).click();

            await expect.poll(async () => columnHeaderValues(page), { timeout: 10000 }).not.toContain('Doing');

            const hiddenItem = page.locator('.octo-board-hidden-item').filter({ hasText: 'Doing' });
            await expect(hiddenItem).toBeVisible({ timeout: 10000 });
            await hiddenItem.locator('.Label').click();
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Show' }).click();

            await columnHeader(page, 'Doing');
        });

        test('can delete a kanban column option', async ({ page }) => {
            await openKanbanBoard(page, 'Delete Column Board');

            const header = await columnHeader(page, 'Done');
            await header.locator('.MenuWrapper button.IconButton').first().click();
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Delete' }).click();

            await expect.poll(async () => columnHeaderValues(page), { timeout: 10000 }).not.toContain('Done');
            expect(await columnHeaderValues(page)).toContain('To Do');
        });

        test('can set a Count calculation on a kanban column', async ({ page }) => {
            const board = await openKanbanBoard(page, 'Calculation Board');
            await createCardViaApi(mattermost, board.boardId, 'Calc One', board.token, { [STATUS_PROPERTY_ID]: board.optionIds['To Do'] });
            await createCardViaApi(mattermost, board.boardId, 'Calc Two', board.token, { [STATUS_PROPERTY_ID]: board.optionIds['To Do'] });
            await page.reload({ waitUntil: 'domcontentloaded' });

            const header = await columnHeader(page, 'To Do');
            await header.locator('.KanbanCalculation button').click({ force: true });
            await header.locator('.KanbanCalculation').getByText('Count', { exact: true }).first().click();

            await expect(header.locator('.KanbanCalculation button')).toContainText('2', { timeout: 10000 });
        });

        test('can filter by a date property being set', async ({ page }) => {
            const board = await openKanbanBoard(page, 'Date Filter Board', [
                { id: 'dateprop', name: 'Due', type: 'date', options: [] },
            ]);
            await createCardViaApi(mattermost, board.boardId, 'Has Date', board.token, {
                dateprop: JSON.stringify({ from: Date.now() }),
            });
            await createCardViaApi(mattermost, board.boardId, 'No Date', board.token);
            await page.reload({ waitUntil: 'domcontentloaded' });
            await expect(page.locator('.KanbanCard').filter({ hasText: 'No Date' })).toBeVisible({ timeout: 15000 });

            await page.locator('.ViewHeader').getByRole('button', { name: 'Filter' }).click();
            await page.getByRole('button', { name: '+ Add filter' }).click();

            const entry = page.locator('.FilterEntry').last();
            await entry.getByRole('button', { name: 'Status' }).click();
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Due' }).click();
            await page.locator('.FilterEntry').last().locator('.MenuWrapper').nth(1).getByRole('button').click();
            await page.locator('.Menu.noselect').getByRole('button', { name: 'is set' }).click();

            await expect(page.locator('.KanbanCard').filter({ hasText: 'No Date' })).toHaveCount(0, { timeout: 10000 });
            await expect(page.locator('.KanbanCard').filter({ hasText: 'Has Date' })).toBeVisible();
        });
    });

    // Runs last: it adds Status options and an active filter on the shared board,
    // which would hide empty-status cards created by the tests above.
    test('filter value with many options stays on one line, ellipsised, with a full tooltip', async ({ page }) => {
        await createBoard(page);

        // Register several Status options so the filter value is long enough to overflow.
        const statuses = ['Not Started', 'In Progress', 'In Review', 'Ready for QA', 'Completed'];
        for (let i = 0; i < statuses.length; i++) {
            await createCardWithStatus(page, `Card ${i + 1}`, statuses[i]);
        }

        // Open the Filter panel and add a filter (defaults to: Status includes …).
        await page.locator('.ViewHeader').getByRole('button', { name: 'Filter' }).click();
        await page.getByRole('button', { name: '+ Add filter' }).click();

        const filterEntry = page.locator('.FilterEntry').last();
        const valueButton = filterEntry.locator('.filterValue button.Button');

        // Switch every Status option on.
        await valueButton.click();
        const optionMenu = page.locator('.Menu.noselect');
        await expect(optionMenu).toBeVisible({ timeout: 5000 });

        // Toggle each option on, waiting for the value label to register the
        // addition before clicking the next. Each onClick reads the current
        // filter values, so clicking faster than the store propagates would
        // drop earlier selections.
        for (const status of statuses) {
            await optionMenu.getByRole('button', { name: status, exact: true }).click();
            await expect(valueButton).toContainText(status, { timeout: 5000 });
        }
        await page.keyboard.press('Escape');

        const expectedLabel = statuses.join(', ');

        // Tooltip: the full list must be exposed via the native title attribute.
        await expect(valueButton).toHaveAttribute('title', expectedLabel, { timeout: 10000 });
        await expect(valueButton).toHaveText(expectedLabel);

        // The label span must be truncated on a single line (not wrapped/sliced).
        const labelSpan = valueButton.locator('span');
        await expect(labelSpan).toHaveCSS('white-space', 'nowrap');
        await expect(labelSpan).toHaveCSS('text-overflow', 'ellipsis');
        await expect(labelSpan).toHaveCSS('overflow', 'hidden');

        // Geometry: no vertical overflow (single line, nothing sliced) but the
        // label does overflow horizontally, which is what makes the ellipsis render.
        const geometry = await valueButton.evaluate((btn) => {
            const span = btn.querySelector('span') as HTMLElement;
            return {
                btnScrollHeight: btn.scrollHeight,
                btnClientHeight: btn.clientHeight,
                spanScrollWidth: span.scrollWidth,
                spanClientWidth: span.clientWidth,
            };
        });
        expect(geometry.btnScrollHeight).toBe(geometry.btnClientHeight);
        // Use a margin so the overflow assertion stays robust against CI font variance.
        expect(geometry.spanScrollWidth).toBeGreaterThan(geometry.spanClientWidth + 20);
    });
});
