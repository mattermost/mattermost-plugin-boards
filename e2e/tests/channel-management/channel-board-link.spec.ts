// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';

const adminUser = 'admin';
const adminPass = 'admin';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

// ─────────────────────────────────────────────────────────────────────────────
// API Helpers
// ─────────────────────────────────────────────────────────────────────────────

async function getTeamId(): Promise<string> {
    const adminClient = await mattermost.getAdminClient();
    const teams = await adminClient.getMyTeams();
    return teams[0]?.id ?? '';
}

/** Create a regular board via API; returns board id. */
async function createBoardViaApi(title: string, token: string): Promise<string> {
    const teamId = await getTeamId();
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        // cardProperties must be [] — Go nil slice serialises as null which crashes
        // getCurrentViewGroupBy when it calls null.find().
        body: JSON.stringify({ teamId, title, type: 'O', cardProperties: [] }),
    });
    if (!resp.ok) throw new Error(`Failed to create board: ${resp.status}`);
    return ((await resp.json()) as { id: string }).id;
}

/** Open the Boards RHS panel via the App Bar button. */
async function openBoardsRHS(page: import('@playwright/test').Page): Promise<void> {
    // In Mattermost 9+, plugin RHS buttons live in the App Bar (far-right icon strip),
    // not the channel header. The button's accessible name is "focalboard" (img alt text).
    await page.getByRole('button', { name: 'focalboard' }).click();
    // Wait for the RHS panel to render
    await expect(page.locator('.focalboard-body').filter({
        has: page.locator('.RHSChannelBoards'),
    })).toBeVisible({ timeout: 15000 });
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Channel-Board Linking', () => {
    test.describe.configure({ timeout: 300000 });

    test('RHS shows empty state when no boards are linked', async ({ page }) => {
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });

        await openBoardsRHS(page);

        await expect(page.locator('.RHSChannelBoards.empty')).toBeVisible({ timeout: 15000 });
        await expect(page.locator('.RHSChannelBoards.empty h2')).toContainText('No boards are linked to');
    });

    test('can link a board to a channel via RHS and see it listed', async ({ page }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = adminClient.getToken();
        const boardTitle = `RHS Link Test ${Date.now()}`;

        // Create a board to link
        await createBoardViaApi(boardTitle, token);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });

        // Open the Boards RHS panel
        await openBoardsRHS(page);

        // Click the "Link boards to..." button in the empty state
        await expect(page.locator('.RHSChannelBoards.empty')).toBeVisible({ timeout: 15000 });
        await page.locator('.RHSChannelBoards.empty button').click();

        // BoardSelector dialog should appear
        await expect(page.locator('.BoardSelector')).toBeVisible({ timeout: 15000 });

        // Search for the board
        await page.locator('.BoardSelector input.searchQuery').fill(boardTitle);

        // Wait for search results
        const boardItem = page.locator('.BoardSelectorItem').filter({ hasText: boardTitle });
        await expect(boardItem).toBeVisible({ timeout: 10000 });

        // Click "Link" on the matching result
        await boardItem.locator('button').filter({ hasText: 'Link' }).click();

        // Confirmation dialog
        await expect(page.locator('.confirmation-dialog-box')).toBeVisible({ timeout: 10000 });
        await page.getByRole('button', { name: 'Yes, link board' }).click();

        // Board should now appear in the RHS list
        await expect(page.locator('.RHSChannelBoards .rhs-boards-list')).toBeVisible({ timeout: 15000 });
        await expect(
            page.locator('.RHSChannelBoards .rhs-boards-list .RHSChannelBoardItem')
                .filter({ hasText: boardTitle }),
        ).toBeVisible({ timeout: 10000 });
    });

    test('can unlink a board from a channel via RHS options menu', async ({ page }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = adminClient.getToken();
        const boardTitle = `RHS Unlink Test ${Date.now()}`;

        // Create a board to link
        await createBoardViaApi(boardTitle, token);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });

        // Open the Boards RHS panel
        await openBoardsRHS(page);

        // Link the board first
        const linkEmptyBtn = page.locator('.RHSChannelBoards.empty button');
        const addBtn = page.locator('.rhs-boards-header button', { hasText: 'Add' });

        // Prefer the empty-state button; fall back to the Add button if already linked boards exist
        const linkBtn = (await linkEmptyBtn.isVisible({ timeout: 3000 }).catch(() => false))
            ? linkEmptyBtn
            : addBtn;
        await linkBtn.click();

        await expect(page.locator('.BoardSelector')).toBeVisible({ timeout: 15000 });
        await page.locator('.BoardSelector input.searchQuery').fill(boardTitle);

        const boardItem = page.locator('.BoardSelectorItem').filter({ hasText: boardTitle });
        await expect(boardItem).toBeVisible({ timeout: 10000 });
        await boardItem.locator('button').filter({ hasText: 'Link' }).click();

        await expect(page.locator('.confirmation-dialog-box')).toBeVisible({ timeout: 10000 });
        await page.getByRole('button', { name: 'Yes, link board' }).click();

        // Wait for board to appear in the list
        const boardListItem = page.locator('.RHSChannelBoardItem').filter({ hasText: boardTitle });
        await expect(boardListItem).toBeVisible({ timeout: 15000 });

        // Hover to reveal the options menu, then click "Unlink board".
        // The MenuWrapper renders menu actions as direct buttons (not .MenuItem wrappers),
        // so target the button by its accessible name.
        await boardListItem.hover();
        await boardListItem.locator('.MenuWrapper button.IconButton').click();
        await page.getByRole('button', { name: 'Unlink board' }).click();

        // After unlink the board should no longer appear
        await expect(boardListItem).not.toBeVisible({ timeout: 10000 });
    });
});
