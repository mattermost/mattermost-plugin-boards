import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import { createCardViaApi, openBoard, seedBoard } from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Card Content', () => {
    test.describe.configure({ timeout: 300000 });

    async function openCardDialog(page: Page, boardTitle: string, cardTitle: string) {
        const board = await seedBoard(mattermost, Users.admin.username, Users.admin.password, boardTitle);
        const cardId = await createCardViaApi(mattermost, board.boardId, cardTitle, board.token);
        await openBoard(page, mattermost, Users.admin.username, Users.admin.password, board, cardId);

        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 15000 });
        return { dialog, board, cardId };
    }

    async function addContent(page: Page, type: 'text' | 'checkbox' | 'divider') {
        const dialog = page.locator('.Dialog.cardDialog');
        await dialog.locator('.CardDetailContentsMenu').getByRole('button', { name: 'Add content' }).click();
        await page.locator('.Menu.noselect').getByRole('button', { name: type }).click();
    }

    test('can add a text description and it persists after reload', async ({ page }) => {
        const { dialog } = await openCardDialog(page, 'Description Board', 'Description Card');

        await addContent(page, 'text');
        const editor = dialog.locator('.MarkdownEditor .public-DraftEditor-content');
        await editor.click();
        await page.keyboard.type('Persisted description text');
        await dialog.locator('.CardDetail .Editable.title:not([disabled])').click();

        await page.reload({ waitUntil: 'domcontentloaded' });
        await expect(page.locator('.Dialog.cardDialog .octo-editor-preview')).toContainText(
            'Persisted description text',
            { timeout: 15000 },
        );
    });

    test('can add a checkbox block and toggle it', async ({ page }) => {
        const { dialog } = await openCardDialog(page, 'Checkbox Board', 'Checkbox Card');

        await addContent(page, 'checkbox');
        const checkboxElement = dialog.locator('.CheckboxElement').first();
        await expect(checkboxElement).toBeVisible({ timeout: 10000 });

        await checkboxElement.locator('.Editable').fill('Buy milk');
        await checkboxElement.locator('.Editable').press('Escape');

        const checkbox = checkboxElement.locator('input[type="checkbox"]');
        await expect(checkbox).not.toBeChecked();
        await checkbox.check();
        await expect(checkbox).toBeChecked();

        await page.reload({ waitUntil: 'domcontentloaded' });
        await expect(
            page.locator('.Dialog.cardDialog .CheckboxElement input[type="checkbox"]').first(),
        ).toBeChecked({ timeout: 15000 });
    });

    test('can add a divider block', async ({ page }) => {
        const { dialog } = await openCardDialog(page, 'Divider Board', 'Divider Card');

        await addContent(page, 'divider');

        await expect(dialog.locator('.DividerElement')).toHaveCount(1, { timeout: 10000 });
    });

    test('can reorder content blocks by dragging', async ({ page }) => {
        const { dialog } = await openCardDialog(page, 'Reorder Board', 'Reorder Card');

        await addContent(page, 'divider');
        await addContent(page, 'checkbox');

        const blocks = dialog.locator('.content-blocks .ContentBlock');
        await expect(blocks).toHaveCount(2, { timeout: 10000 });
        await expect(blocks.first().locator('.DividerElement')).toBeVisible();

        const firstBlock = blocks.first();
        const secondBlock = blocks.nth(1);
        await secondBlock.hover();
        await secondBlock.locator('.dnd-handle').dragTo(firstBlock, { targetPosition: { x: 40, y: 2 } });

        await expect(blocks.first().locator('.CheckboxElement')).toBeVisible({ timeout: 10000 });
    });

    test('can upload an attachment with the Attach button', async ({ page }) => {
        const { dialog } = await openCardDialog(page, 'Attachment Board', 'Attachment Card');

        const chooserPromise = page.waitForEvent('filechooser');
        await dialog.getByRole('button', { name: 'Attach' }).click();
        const chooser = await chooserPromise;
        await chooser.setFiles({
            name: 'e2e-attachment.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('hello from e2e'),
        });

        await expect(dialog.locator('.Attachment .FileElement')).toContainText('e2e-attachment', { timeout: 20000 });
        await expect(dialog.locator('.Attachment .attachment-title')).toContainText('(1)');
    });
});
