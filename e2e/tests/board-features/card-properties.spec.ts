import { test, expect, type Page, type Locator } from '@playwright/test';

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

test.describe('Card Properties', () => {
    test.describe.configure({ timeout: 300000 });

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

    async function createNamedCard(page: Page, cardName: string): Promise<void> {
        // The "New" button is a div.ButtonWithMenu (not a <button>); click its text part.
        // .button-text contains the "New" label; .button-dropdown is the arrow (stopPropagation).
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

    async function openCard(page: Page, cardName: string): Promise<Locator> {
        await page.locator('.KanbanCard').filter({ hasText: cardName }).click();
        const dialog = page.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });
        return dialog;
    }

    /**
     * Click "Add a property", pick a type, then dismiss the auto-opened
     * property-name rename popup by pressing Enter on its input.
     *
     * Type names visible in the Focalboard menu (exact text, no hyphens):
     *   Text, Number, Email, Phone, URL, Select, Multi select, Date,
     *   Person, Multi person, Checkbox
     */
    async function addProperty(page: Page, dialog: Locator, typeName: string): Promise<void> {
        await dialog.locator('.add-property').click();

        const typeMenu = page.locator('.Menu.noselect');
        await expect(typeMenu).toBeVisible({ timeout: 5000 });
        await typeMenu.getByRole('button', { name: typeName, exact: true }).click();

        // Focalboard auto-opens a property-name rename popup after adding.
        // Press Enter on its input to confirm the default name and close the popup.
        const namePopupInput = page.locator('.Menu.noselect input');
        const namePopupVisible = await namePopupInput
            .waitFor({ state: 'visible', timeout: 2000 })
            .then(() => true)
            .catch(() => false);

        if (namePopupVisible) {
            await namePopupInput.press('Enter');
            // Wait for the popup to close
            await expect(namePopupInput).not.toBeVisible({ timeout: 3000 });
        }
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Text property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a text property and set a value', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'TextProp Card');
        const dialog = await openCard(page, 'TextProp Card');

        await addProperty(page, dialog, 'Text');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        const input = propRow.locator('input, [contenteditable="true"]').last();
        await input.fill('Hello World');
        await input.press('Enter');

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Number property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a number property and set a value', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'NumberProp Card');
        const dialog = await openCard(page, 'NumberProp Card');

        await addProperty(page, dialog, 'Number');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        const input = propRow.locator('input, [contenteditable="true"]').last();
        await input.fill('42');
        await input.press('Enter');

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Select property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a select property and choose an option', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'SelectProp Card');
        const dialog = await openCard(page, 'SelectProp Card');

        await addProperty(page, dialog, 'Select');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        // CreatableSelect: type a new option name and press Enter to create it
        const selectInput = page.locator('.ValueSelector input').last();
        await selectInput.fill('Option A');
        await selectInput.press('Enter');

        // The property value cell should now contain the option text
        await expect(propRow.locator('.octo-propertyvalue')).toContainText('Option A', { timeout: 5000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Multi-select property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a multi-select property and choose multiple options', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'MultiSelectProp Card');
        const dialog = await openCard(page, 'MultiSelectProp Card');

        // Note: the menu item text is "Multi select" (space, no hyphen)
        await addProperty(page, dialog, 'Multi select');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        // onCreateValue is async (API call). Create each tag one at a time,
        // waiting for the chip to appear before creating the next one to
        // avoid the race condition where the second call captures stale values.
        const selectInput = page.locator('.ValueSelector input').last();
        await selectInput.fill('Tag One');
        await selectInput.press('Enter');

        // Wait for Tag One chip to appear in the selected value area
        await expect(page.locator('.ValueSelector__multi-value').first()).toBeVisible({ timeout: 10000 });

        await selectInput.fill('Tag Two');
        await selectInput.press('Enter');

        // Wait for Tag Two chip to appear (now there should be two chips)
        await expect(page.locator('.ValueSelector__multi-value').nth(1)).toBeVisible({ timeout: 10000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Date property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a date property and set to today', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'DateProp Card');
        const dialog = await openCard(page, 'DateProp Card');

        await addProperty(page, dialog, 'Date');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        // Date picker should open
        const datePicker = page.locator('.DateRange');
        await expect(datePicker).toBeVisible({ timeout: 5000 });

        // Click today's highlighted day cell
        const todayCell = datePicker.locator('.DayPicker-Day--today');
        await expect(todayCell).toBeVisible({ timeout: 5000 });
        await todayCell.click();

        // Close the date picker by clicking the active title textarea.
        // There are two .Editable.title elements (EditableArea + EditableAreaReference);
        // use .EditableArea.title to target only the active (non-disabled) one.
        await dialog.locator('.EditableArea.title').click({ force: true });

        // The value cell should now show a date (no longer empty)
        await expect(propRow.locator('.octo-propertyvalue')).not.toHaveText(/^$/);

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Person property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a person property and assign a user', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'PersonProp Card');
        const dialog = await openCard(page, 'PersonProp Card');

        await addProperty(page, dialog, 'Person');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        // PersonSelector uses react-select/async with defaultOptions=true, so options appear
        // immediately without typing. In react-select v5 the input has class react-select__input
        // (not a wrapper div), so just click the first listed option directly.
        const option = page.locator('.react-select__option').first();
        await expect(option).toBeVisible({ timeout: 5000 });
        await option.click();

        // Value cell should no longer be empty
        await expect(propRow.locator('.octo-propertyvalue')).not.toHaveText(/^Empty$/i, { timeout: 5000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Multi-person property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a multi-person property and assign a user', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'MultiPersonProp Card');
        const dialog = await openCard(page, 'MultiPersonProp Card');

        // Note: the menu item text is "Multi person" (space, no hyphen)
        await addProperty(page, dialog, 'Multi person');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        // Same pattern as Person: defaultOptions=true so options appear immediately
        const option = page.locator('.react-select__option').first();
        await expect(option).toBeVisible({ timeout: 5000 });
        await option.click();

        await expect(propRow.locator('.octo-propertyvalue')).not.toHaveText(/^Empty$/i, { timeout: 5000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Checkbox property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a checkbox property and toggle it on', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'CheckboxProp Card');
        const dialog = await openCard(page, 'CheckboxProp Card');

        await addProperty(page, dialog, 'Checkbox');

        // Checkbox renders its Switch directly inside .octo-propertyrow (no .octo-propertyvalue wrapper).
        const propRow = dialog.locator('.octo-propertyrow').last();
        const switchEl = propRow.locator('.Switch');

        // Verify the switch is off initially
        await expect(switchEl).toBeVisible({ timeout: 5000 });
        const isAlreadyOn = await switchEl.evaluate((el) => el.classList.contains('on'));
        expect(isAlreadyOn).toBe(false);

        // Click the Switch to turn it on
        await switchEl.click();
        await expect(switchEl).toHaveClass(/\bon\b/, { timeout: 3000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Email property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add an email property and set a value', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'EmailProp Card');
        const dialog = await openCard(page, 'EmailProp Card');

        await addProperty(page, dialog, 'Email');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        const input = propRow.locator('input, [contenteditable="true"]').last();
        await input.fill('test@example.com');
        await input.press('Enter');

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // URL property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a URL property and set a value', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'URLProp Card');
        const dialog = await openCard(page, 'URLProp Card');

        await addProperty(page, dialog, 'URL');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        const input = propRow.locator('input, [contenteditable="true"]').last();
        await input.fill('https://mattermost.com');
        await input.press('Enter');

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Phone property
    // ─────────────────────────────────────────────────────────────────────────

    test('can add a phone property and set a value', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'PhoneProp Card');
        const dialog = await openCard(page, 'PhoneProp Card');

        await addProperty(page, dialog, 'Phone');

        const propRow = dialog.locator('.octo-propertyrow').last();
        await propRow.locator('.octo-propertyvalue').click();

        const input = propRow.locator('input, [contenteditable="true"]').last();
        await input.fill('+1-555-123-4567');
        await input.press('Enter');

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Status change → card moves to new kanban group
    // ─────────────────────────────────────────────────────────────────────────

    test('changing status creates new group and moves card to it', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'StatusMove Card');

        const dialog = await openCard(page, 'StatusMove Card');

        // Click the Status property value to open its ValueSelector
        await dialog.locator('.octo-propertyvalue').first().click();

        const selectInput = page.locator('.ValueSelector input').last();
        await selectInput.fill('Shipped');
        await selectInput.press('Enter');

        // Wait for the "Shipped" label to appear in the Status property row before closing.
        // This ensures the async insertPropertyOption + changePropertyValue chain has completed.
        await expect(dialog.locator('.octo-propertyvalue').first().locator('.Label')).toContainText('Shipped', { timeout: 10000 });

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });

        // A new "Shipped" column header should appear in the kanban board.
        // The column title is an Editable <input> — match by DOM value property via JS poll.
        await expect.poll(
            async () => {
                const values = await page
                    .locator('.KanbanColumnHeader input.Editable')
                    .evaluateAll((els) => els.map((el) => (el as HTMLInputElement).value));
                return values.includes('Shipped');
            },
            { timeout: 15000 },
        ).toBe(true);

        // The card should still be visible (now in the Shipped column)
        await expect(page.locator('.KanbanCard').filter({ hasText: 'StatusMove Card' })).toBeVisible({ timeout: 10000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Property value visible on closed kanban card face
    // ─────────────────────────────────────────────────────────────────────────

    test('select property value is visible on kanban card face when toggled on', async ({ page }) => {
        await createBoard(page);
        await createNamedCard(page, 'FaceProp Card');

        const dialog = await openCard(page, 'FaceProp Card');
        await addProperty(page, dialog, 'Select');

        const propRow = dialog.locator('.octo-propertyrow').last();

        // Capture the property's default name (will be "Select" for a new Select property)
        const propName = (await propRow.locator('.octo-propertyname').innerText()).trim();

        await propRow.locator('.octo-propertyvalue').click();

        const selectInput = page.locator('.ValueSelector input').last();
        await selectInput.fill('Visible');
        await selectInput.press('Enter');

        await dialog.locator('.dialog__close').click();
        await expect(dialog).not.toBeVisible({ timeout: 5000 });

        // Open Properties panel and enable the new property if it is currently off
        await page.locator('.ViewHeader').getByRole('button', { name: 'Properties', exact: true }).click();
        const propertiesMenu = page.locator('.Menu.noselect');
        await expect(propertiesMenu).toBeVisible({ timeout: 5000 });

        // Properties are listed in board order; the menu always ends with "Comments and description".
        // Our new property is the second-to-last SwitchOption.
        const allSwitches = propertiesMenu.locator('.SwitchOption');
        const count = await allSwitches.count();
        const propSwitch = allSwitches.nth(count - 2).locator('.Switch');
        const isOn = await propSwitch.evaluate((el) => el.classList.contains('on'));
        if (!isOn) {
            await propSwitch.click();
        }

        await page.locator('.ViewHeader').getByRole('button', { name: 'Properties', exact: true }).click();
        await expect(propertiesMenu).not.toBeVisible({ timeout: 5000 });

        // The "Visible" label should now appear on the closed kanban card face
        await expect(
            page.locator('.KanbanCard').filter({ hasText: 'FaceProp Card' }).locator('.Label'),
        ).toContainText('Visible', { timeout: 10000 });
    });
});
