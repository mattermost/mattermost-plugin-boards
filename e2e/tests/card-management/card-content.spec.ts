import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import {
    createCardViaApi,
    dragAndDrop,
    getBlocksViaApi,
    openBoard,
    seedBoard,
    waitForBlockPatch,
    type SeededBoard,
} from 'helpers/boards';

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

    const contentBlocks = (page: Page) => page.locator('.Dialog.cardDialog .content-blocks .ContentBlock');

    async function addContent(page: Page, type: 'text' | 'checkbox' | 'divider') {
        const dialog = page.locator('.Dialog.cardDialog');
        const countBefore = await contentBlocks(page).count();
        await dialog.locator('.CardDetailContentsMenu').getByRole('button', { name: 'Add content' }).click();
        await page.locator('.Menu.noselect').getByRole('button', { name: type }).click();
        await expect(contentBlocks(page)).toHaveCount(countBefore + 1, { timeout: 10000 });
    }

    test('can add a text description and it persists after reload', async ({ page }) => {
        const { dialog } = await openCardDialog(page, 'Description Board', 'Description Card');

        await addContent(page, 'text');
        const textBlock = contentBlocks(page).first();
        await textBlock.locator('.octo-editor-preview').click();
        await textBlock.locator('.MarkdownEditor .public-DraftEditor-content').click();
        await page.keyboard.type('Persisted description text');
        const saved = waitForBlockPatch(page, (patch) => patch.title === 'Persisted description text');
        await dialog.locator('.CardDetail .Editable.title:not([disabled])').click();
        await saved;

        await page.reload({ waitUntil: 'domcontentloaded' });
        await expect(contentBlocks(page).first().locator('.octo-editor-preview')).toContainText(
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
        const titleSaved = waitForBlockPatch(page, (patch) => patch.title === 'Buy milk');
        await checkboxElement.locator('.Editable').press('Escape');
        await titleSaved;

        const checkbox = checkboxElement.locator('input[type="checkbox"]');
        await expect(checkbox).not.toBeChecked();
        const toggleSaved = waitForBlockPatch(page, (patch) => patch.updatedFields?.value === true);
        // The change handler calls preventDefault and re-renders as checked, which check() reads as a no-op.
        await checkbox.click();
        await toggleSaved;
        await expect(checkbox).toBeChecked();

        await page.reload({ waitUntil: 'domcontentloaded' });
        await expect(
            page.locator('.Dialog.cardDialog .CheckboxElement input[type="checkbox"]').first(),
        ).toBeChecked({ timeout: 15000 });
    });

    /** The block types listed in the card's saved contentOrder, in order. */
    async function savedContentTypes(board: SeededBoard, cardId: string): Promise<string[]> {
        const blocks = await getBlocksViaApi(mattermost, board.boardId, board.token);
        const card = blocks.find((b) => b.id === cardId)!;
        const contentOrder = (card.fields.contentOrder ?? []) as Array<string | string[]>;
        return contentOrder.flat().map((id) => blocks.find((b) => b.id === id)?.type ?? 'missing');
    }

    test('can add a divider block', async ({ page }) => {
        const { dialog, board, cardId } = await openCardDialog(page, 'Divider Board', 'Divider Card');

        await addContent(page, 'divider');

        await expect(dialog.locator('.DividerElement')).toHaveCount(1, { timeout: 10000 });
        await expect.poll(() => savedContentTypes(board, cardId), { timeout: 10000 }).toEqual(['divider']);
    });

    test('can reorder content blocks by dragging', async ({ page }) => {
        const { dialog, board, cardId } = await openCardDialog(page, 'Reorder Board', 'Reorder Card');

        await addContent(page, 'divider');
        await addContent(page, 'checkbox');

        const blocks = contentBlocks(page);
        await expect(blocks).toHaveCount(2);
        await expect(blocks.first().locator('.DividerElement')).toBeVisible();
        await expect.poll(() => savedContentTypes(board, cardId), { timeout: 10000 }).toEqual(['divider', 'checkbox']);

        // Row reordering is handled by the drop zone above each row, not the block itself.
        const aboveFirstRow = dialog.locator('.octo-content > div > .addToRow').first();
        const secondBlock = blocks.nth(1);
        await secondBlock.hover();
        await dragAndDrop(page, secondBlock.locator('.dnd-handle'), aboveFirstRow);

        await expect(blocks.first().locator('.CheckboxElement')).toBeVisible({ timeout: 10000 });
        await expect.poll(() => savedContentTypes(board, cardId), { timeout: 10000 }).toEqual(['checkbox', 'divider']);
    });

    test('can upload an attachment with the Attach button', async ({ page }) => {
        const { dialog, board } = await openCardDialog(page, 'Attachment Board', 'Attachment Card');

        const chooserPromise = page.waitForEvent('filechooser');
        await dialog.getByRole('button', { name: 'Attach' }).click();
        const chooser = await chooserPromise;
        const attachmentSaved = page.waitForResponse((response) =>
            response.request().method() === 'POST' &&
            new URL(response.url()).pathname.endsWith(`/boards/${board.boardId}/blocks`) &&
            response.ok());
        await chooser.setFiles({
            name: 'e2e-attachment.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('hello from e2e'),
        });
        await attachmentSaved;
        await expect(dialog.locator('.Attachment .fileElement-file-uploading')).toHaveCount(0, { timeout: 20000 });

        await page.reload({ waitUntil: 'domcontentloaded' });
        const reloadedDialog = page.locator('.Dialog.cardDialog');
        await expect(reloadedDialog.locator('.Attachment .FileElement')).toContainText('e2e-attachment', { timeout: 20000 });
        await expect(reloadedDialog.locator('.Attachment .attachment-title')).toContainText('(1)');
    });
});
