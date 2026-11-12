import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { Users } from 'helpers/users';
import { createCardViaApi, openBoard, openKanbanCardMenu, seedBoard } from 'helpers/boards';

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

test.describe('Card Management', () => {
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

            // New board: verify untitled placeholder shows on the right, then name it.
            await expect(page.locator('.ViewTitle .Editable.title')).toHaveAttribute('placeholder', /[Uu]ntitled/, { timeout: 10000 });
            const titleInput = page.locator('.ViewTitle .Editable');
            await titleInput.click();
            await titleInput.fill('Test Board');
            await titleInput.press('Enter');

            // Verify the name now appears in the sidebar.
            await expect(page.locator('.octo-sidebar-list')).toContainText('Test Board', { timeout: 10000 });
        }

        await expect(boardComponent).toBeVisible({ timeout: 15000 });
    }

    /**
     * Create a new card and give it a name. Returns the card name used.
     */
    async function createNamedCard(page: Page, cardName: string): Promise<void> {
        await page.getByRole('button', { name: 'New' }).click();

        // Clicking "New" opens the card dialog immediately — no need to click the card.
        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        const titleField = dialog.locator('.CardDetail .Editable.title:not([disabled])');
        await titleField.click();
        await titleField.fill(cardName);

        // Close the dialog to persist the name
        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });

        // Verify the card shows the name in the kanban
        await expect(page.locator('.KanbanCard').filter({ hasText: cardName })).toBeVisible({ timeout: 10000 });
    }

    test('can create a new card', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'New Card One');
    });

    test('can open a card and view its dialog', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'Dialog Test Card');

        // Re-open the card to verify the dialog appears
        await page.locator('.KanbanCard').filter({ hasText: 'Dialog Test Card' }).click();
        await expect(page.locator('.Dialog.cardDialog')).toBeVisible({ timeout: 10000 });
    });

    test('can close the card dialog', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'Close Dialog Card');

        // Re-open and close the dialog
        await page.locator('.KanbanCard').filter({ hasText: 'Close Dialog Card' }).click();
        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    test('can edit a card title', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'Original Card Title');

        // Re-open the card and rename it
        await page.locator('.KanbanCard').filter({ hasText: 'Original Card Title' }).click();
        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        const titleField = dialog.locator('.CardDetail .Editable.title:not([disabled])');
        await titleField.click();
        await titleField.fill('My Test Card');

        await dialog.locator('.dialog__close').click();
        await expect(page.locator('.KanbanCard').filter({ hasText: 'My Test Card' })).toBeVisible({ timeout: 10000 });
    });

    test.describe('seeded board actions', () => {
        async function openSeededBoard(page: Page, boardTitle: string, cardTitle: string) {
            const board = await seedBoard(mattermost, Users.admin.username, Users.admin.password, boardTitle);
            const cardId = await createCardViaApi(mattermost, board.boardId, cardTitle, board.token);
            await openBoard(page, mattermost, Users.admin.username, Users.admin.password, board);
            await expect(page.locator('.KanbanCard').filter({ hasText: cardTitle })).toBeVisible({ timeout: 10000 });
            return { board, cardId };
        }

        test('can delete a card from the card actions menu', async ({ page }) => {
            await openSeededBoard(page, 'Delete Card Board', 'Card To Delete');

            await openKanbanCardMenu(page, 'Card To Delete');
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Delete' }).click();
            await page.locator('.confirmation-dialog-box').getByRole('button', { name: 'Delete' }).click();

            await expect(page.locator('.KanbanCard').filter({ hasText: 'Card To Delete' })).toHaveCount(0, { timeout: 10000 });
        });

        test('can duplicate a card', async ({ page }) => {
            await openSeededBoard(page, 'Duplicate Card Board', 'Card To Duplicate');

            await openKanbanCardMenu(page, 'Card To Duplicate');
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Duplicate' }).click();

            const dialog = page.locator('.Dialog.cardDialog');
            await expect(dialog).toBeVisible({ timeout: 10000 });
            await dialog.locator('.dialog__close').click();

            await expect(page.locator('.KanbanCard').filter({ hasText: 'Card To Duplicate' })).toHaveCount(2, { timeout: 10000 });
        });

        test('can copy a card link and see the confirmation message', async ({ page }) => {
            await openSeededBoard(page, 'Copy Link Board', 'Card To Link');

            await openKanbanCardMenu(page, 'Card To Link');
            await page.locator('.Menu.noselect').getByRole('button', { name: 'Copy link' }).click();

            await expect(page.locator('.FlashMessages')).toContainText('Copied!', { timeout: 5000 });
        });

        test('deep link to a card URL opens the card dialog', async ({ page }) => {
            const board = await seedBoard(mattermost, Users.admin.username, Users.admin.password, 'Deep Link Board');
            const cardId = await createCardViaApi(mattermost, board.boardId, 'Deep Linked Card', board.token);

            await openBoard(page, mattermost, Users.admin.username, Users.admin.password, board, cardId);

            const dialog = page.locator('.Dialog.cardDialog');
            await expect(dialog).toBeVisible({ timeout: 15000 });
            await expect(dialog.locator('.CardDetail .Editable.title:not([disabled])')).toHaveValue('Deep Linked Card');
        });
    });
});
