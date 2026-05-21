import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';

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

test.describe('Board Views', () => {
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

        // Wait for the page to settle into one of two stable states, then decide.
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
     * Open the View menu dropdown and add a new view of the given type.
     *
     * "Add view" lives inside the ViewMenu (the ▾ dropdown next to the view
     * name in the header) — it is NOT a standalone button in the toolbar.
     *
     * SubMenuOption.onClick toggles isOpen. We use dispatchEvent (no pointer
     * movement) so the hover/HoveringContext mechanism never fires, which would
     * otherwise open-then-immediately-close the sub-menu on a real click().
     */
    async function addView(page: Page, viewType: 'Board' | 'Table' | 'Gallery'): Promise<void> {
        // Open the view-menu dropdown
        await page.locator('.ViewHeader').getByRole('button', { name: 'View menu' }).click();

        // dispatchEvent fires onClick without pointer movement, toggling isOpen
        // from false → true without triggering the hover mechanism.
        await page.locator('#__addView').dispatchEvent('click');

        // Sub-menu items are DOM children of #__addView when isOpen=true.
        // TextOption renders role="button" aria-label={name} but does NOT set a DOM id,
        // so getByRole is the correct selector here.
        await page.locator('#__addView').getByRole('button', { name: viewType }).dispatchEvent('click');
    }

    test('default view is board (kanban)', async ({ page }) => {
        await createBoard(page);
        await expect(page.locator('.Kanban')).toBeVisible({ timeout: 10000 });
    });

    test('can switch to table view', async ({ page }) => {
        await createBoard(page);
        await addView(page, 'Table');
        await expect(page.locator('.Table')).toBeVisible({ timeout: 10000 });
    });

    test('can switch to gallery view', async ({ page }) => {
        await createBoard(page);
        await addView(page, 'Gallery');
        await expect(page.locator('.Gallery')).toBeVisible({ timeout: 10000 });
    });

    test('can switch back to board view from table view', async ({ page }) => {
        await createBoard(page);
        await addView(page, 'Table');
        await expect(page.locator('.Table')).toBeVisible({ timeout: 10000 });

        // Click the original "Board view" tab in the sidebar.
        // Both classes are on the same element, so use a compound selector (no space).
        await page.locator('.SidebarBoardItem.sidebar-view-item').filter({ hasText: 'Board view' }).click();
        await expect(page.locator('.Kanban')).toBeVisible({ timeout: 10000 });
    });

    test('table view shows add row button', async ({ page }) => {
        await createBoard(page);
        await addView(page, 'Table');
        await expect(page.locator('.Table')).toBeVisible({ timeout: 10000 });
        await expect(page.locator('.octo-table-footer')).toBeVisible({ timeout: 10000 });
    });
});
