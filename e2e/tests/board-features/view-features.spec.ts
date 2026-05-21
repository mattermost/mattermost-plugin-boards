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
     */
    async function createNamedCard(page: Page, cardName: string): Promise<void> {
        await page.getByRole('button', { name: 'New' }).click();

        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        const titleField = dialog.locator('.CardDetail .Editable.title:not([disabled])');
        await titleField.click();
        await titleField.fill(cardName);

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });

        await expect(page.locator('.KanbanCard').filter({ hasText: cardName })).toBeVisible({ timeout: 10000 });
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
});
