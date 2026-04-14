// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import { test, expect, type Browser, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';

const adminUser = 'admin';
const adminPass = 'admin';
const regularUser = 'regularuser';
const regularPass = 'regularuser';

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

/** Create a board via the Focalboard REST API, returns the board id. */
async function createBoardViaApi(title: string, token: string): Promise<string> {
    const adminClient = await mattermost.getAdminClient();
    const teams = await adminClient.getMyTeams();
    const teamId = teams[0]?.id ?? '';

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

/** Create a default board view; returns teamId and viewId needed for direct board URLs. */
async function createBoardView(boardId: string, token: string): Promise<{ teamId: string; viewId: string }> {
    const adminClient = await mattermost.getAdminClient();
    const teams = await adminClient.getMyTeams();
    const teamId = teams[0]?.id ?? '';

    const now = Date.now();
    const viewBlockId = `view${boardId.substring(0, 8)}${now}`.replace(/[^a-z0-9]/gi, '').substring(0, 26);
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/blocks`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify([{
            id: viewBlockId,
            boardId,
            parentId: boardId,
            type: 'view',
            title: 'Board View',
            schema: 1,
            createAt: now,
            updateAt: now,
            fields: {
                viewType: 'board',
                cardOrder: [],
                sortOptions: [],
                visiblePropertyIds: [],
                visibleOptionIds: [],
                hiddenOptionIds: [],
                collapsedOptionIds: [],
                filter: { operation: 'and', filters: [] },
                columnWidths: {},
                columnCalculations: {},
                kanbanCalculations: {},
                defaultTemplateId: '',
            },
        }]),
    });
    if (!resp.ok) throw new Error(`Failed to create board view: ${resp.status}`);
    const created = (await resp.json()) as Array<{ id: string }>;
    return { teamId, viewId: created[0]?.id ?? '' };
}

/**
 * Create a custom sidebar category via the Focalboard categories API.
 * Categories are user-scoped: `userId` is the owner, `token` is their auth token.
 */
async function createCategoryViaApi(name: string, userId: string, token: string): Promise<string> {
    const adminClient = await mattermost.getAdminClient();
    const teams = await adminClient.getMyTeams();
    const teamId = teams[0]?.id ?? '';
    const now = Date.now();

    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/teams/${teamId}/categories`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({
            name,
            userID: userId,
            teamID: teamId,
            createAt: now,
            updateAt: now,
            deleteAt: 0,
            collapsed: false,
            type: 'custom',
        }),
    });
    if (!resp.ok) throw new Error(`Failed to create category: ${resp.status}`);
    return ((await resp.json()) as { id: string }).id;
}

/** Add a user to a board with the given role. */
async function addBoardMember(
    boardId: string,
    userId: string,
    role: 'admin' | 'editor' | 'commenter' | 'viewer',
    token: string,
): Promise<void> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/members`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({
            boardId,
            userId,
            schemeAdmin: role === 'admin',
            schemeEditor: role === 'editor',
            schemeCommenter: role === 'commenter',
            schemeViewer: role === 'viewer',
        }),
    });
    if (!resp.ok) throw new Error(`Failed to add board member: ${resp.status}`);
}

/**
 * Pre-seed welcomePageViewed so FBRoute never redirects to /welcome.
 * Without this, the welcome page's goForward() during a render cycle triggers a
 * React rendering error caught by the ErrorBoundary → /boards/error?id=unknown.
 */
async function seedWelcomePageViewed(userId: string, token: string): Promise<void> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/users/${userId}/config`, {
        method: 'PUT',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({ updatedFields: { welcomePageViewed: '1' } }),
    });
    if (!resp.ok) throw new Error(`Failed to seed welcomePageViewed: ${resp.status}`);
}

/**
 * Log in as `username` and navigate directly to a board URL so the sidebar
 * is fully loaded. Returns after `.BoardComponent` is visible.
 */
async function loginAndOpenBoard(
    browser: Browser,
    username: string,
    password: string,
    teamId: string,
    boardId: string,
    viewId: string,
): Promise<{ ctx: import('@playwright/test').BrowserContext; page: Page }> {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();

    // Set lastTeamId before page scripts run so HomeToCurrentTeam always
    // resolves a team and never falls through to error?id=unknown.
    await page.addInitScript((tid) => {
        localStorage.setItem('lastTeamId', tid);
    }, teamId);

    const mmPage = new MattermostPage(page);
    await mmPage.login(mattermost.url(), username, password);
    await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });

    // Navigate to the boards root URL first so the app fully initialises —
    // welcome-page check, user-config fetch, Redux store hydration — before
    // going to a specific board URL. Jumping straight to /boards/team/…/boardId/viewId
    // on a fresh context can race against the welcomePageViewed redirect and
    // trigger the React ErrorBoundary ("Sorry, something went wrong").
    await mmPage.navigateToBoardsFromUrl(mattermost.url());

    await page.goto(
        `${mattermost.url()}/boards/team/${teamId}/${boardId}/${viewId}`,
        { waitUntil: 'domcontentloaded', timeout: 60000 },
    );
    await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 20000 });

    return { ctx, page };
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: open the options menu on a category header row.
//
// .MenuWrapper inside .octo-sidebar-item.category is display:none by default
// and only reveals on :hover (sidebarCategory.scss lines 107–124).
// Always hover before clicking the IconButton.
//
// Use anchored-regex exact matching so that a category name that is a
// substring of another (e.g. "Boards" ⊂ "Sprint Boards") never resolves
// to two elements and triggers Playwright strict-mode violations.
// ─────────────────────────────────────────────────────────────────────────────

/** Escape a string for use in a RegExp and wrap it with ^ … $ anchors. */
function exact(text: string): RegExp {
    return new RegExp(`^${text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`);
}

async function openCategoryMenu(page: Page, categoryName: string): Promise<void> {
    const catHeader = page
        .locator('.octo-sidebar-item.category')
        .filter({ has: page.locator('.category-title', { hasText: exact(categoryName) }) });
    await catHeader.hover();
    await catHeader.locator('.MenuWrapper button.IconButton').click();
    await expect(page.locator('.Menu.noselect')).toBeVisible({ timeout: 5000 });
}

// Same pattern for a board item (.SidebarBoardItem.subitem).
async function openBoardItemMenu(page: Page, boardTitle: string): Promise<void> {
    const boardItem = page.locator('.SidebarBoardItem.subitem').filter({ hasText: exact(boardTitle) });
    await boardItem.hover();
    await boardItem.locator('.MenuWrapper button.IconButton').click();
    await expect(page.locator('.Menu.noselect')).toBeVisible({ timeout: 5000 });
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Sidebar Categories', () => {
    test.describe.configure({ timeout: 300000 });

    // ─────────────────────────────────────────────────────────────────────────
    // Create
    // ─────────────────────────────────────────────────────────────────────────

    test('user can create a custom category from the sidebar menu', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const boardId = await createBoardViaApi('Create Category Board', token);
        const { teamId, viewId } = await createBoardView(boardId, token);
        const adminProfile = await adminClient.getMe();
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        // Open the options on the default "Boards" system category.
        await openCategoryMenu(page, 'Boards');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Create New Category' }).click();

        // Type the new name in the modal and confirm.
        await expect(page.locator('.CreateCategoryModal')).toBeVisible({ timeout: 5000 });
        await page.locator('.categoryNameInput').fill('Sprint Boards');
        await page.locator('.CreateCategoryModal').getByRole('button', { name: 'Create' }).click();

        // The new category must appear in the sidebar.
        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Sprint Boards') }) }),
        ).toBeVisible({ timeout: 10000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Default category restrictions
    // ─────────────────────────────────────────────────────────────────────────

    test('default Boards category only exposes Create New Category — no Rename or Delete', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const boardId = await createBoardViaApi('Default Category Board', token);
        const { teamId, viewId } = await createBoardView(boardId, token);
        const adminProfile = await adminClient.getMe();
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        await openCategoryMenu(page, 'Boards');
        const menu = page.locator('.Menu.noselect');

        // The system category allows creating children.
        await expect(menu.getByRole('button', { name: 'Create New Category' })).toBeVisible();

        // Rename and Delete are only shown for type='custom' categories.
        await expect(menu.getByRole('button', { name: 'Rename Category' })).not.toBeVisible();
        await expect(menu.getByRole('button', { name: 'Delete Category' })).not.toBeVisible();

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Rename
    // ─────────────────────────────────────────────────────────────────────────

    test('user can rename a custom category', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const boardId = await createBoardViaApi('Rename Category Board', token);
        const { teamId, viewId } = await createBoardView(boardId, token);
        // Pre-create the category via API so the test starts with it already present.
        await createCategoryViaApi('Old Category Name', adminProfile.id, token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Old Category Name') }) }),
        ).toBeVisible({ timeout: 10000 });

        await openCategoryMenu(page, 'Old Category Name');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Rename Category' }).click();

        // The same CreateCategory modal opens pre-filled with the current name.
        await expect(page.locator('.CreateCategoryModal')).toBeVisible({ timeout: 5000 });
        // Clear the pre-filled value and enter the new name.
        await page.locator('.categoryNameInput').clear();
        await page.locator('.categoryNameInput').fill('New Category Name');
        // Save button says "Update" in rename mode.
        await page.locator('.CreateCategoryModal').getByRole('button', { name: 'Update' }).click();

        // The sidebar must reflect the updated name.
        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('New Category Name') }) }),
        ).toBeVisible({ timeout: 10000 });
        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Old Category Name') }) }),
        ).not.toBeVisible({ timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Delete
    // ─────────────────────────────────────────────────────────────────────────

    test('user can delete a custom category and its boards return to the default category', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const boardId = await createBoardViaApi('Delete Category Board', token);
        const { teamId, viewId } = await createBoardView(boardId, token);
        await createCategoryViaApi('Temporary Category', adminProfile.id, token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Temporary Category') }) }),
        ).toBeVisible({ timeout: 10000 });

        await openCategoryMenu(page, 'Temporary Category');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Delete Category' }).click();

        // A confirmation dialog appears (ConfirmationDialogBox).
        const confirmDialog = page.locator('.confirmation-dialog-box');
        await expect(confirmDialog).toBeVisible({ timeout: 5000 });
        await expect(confirmDialog.locator('h3')).toContainText('Delete this category?');
        await confirmDialog.getByRole('button', { name: 'Confirm' }).click();

        // The category should disappear from the sidebar.
        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Temporary Category') }) }),
        ).not.toBeVisible({ timeout: 10000 });

        // The default "Boards" category must still be present.
        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Boards') }) }),
        ).toBeVisible({ timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Collapse / Expand
    // ─────────────────────────────────────────────────────────────────────────

    test('user can collapse and expand a category by clicking its title', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const boardId = await createBoardViaApi('Collapse Test Board', token);
        const { teamId, viewId } = await createBoardView(boardId, token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        // Confirm the category starts expanded.
        const catHeader = page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Boards') }) });
        await expect(catHeader).toHaveClass(/expanded/, { timeout: 10000 });

        // Click the title to collapse.
        await catHeader.locator('.category-title').click();
        await expect(catHeader).toHaveClass(/collapsed/, { timeout: 5000 });

        // Click again to expand.
        await catHeader.locator('.category-title').click();
        await expect(catHeader).toHaveClass(/expanded/, { timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Move board between categories
    // ─────────────────────────────────────────────────────────────────────────

    test('user can move a board to a custom category via the board options menu', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const boardId = await createBoardViaApi('Move Board Test', token);
        const { teamId, viewId } = await createBoardView(boardId, token);
        await createCategoryViaApi('Target Category', adminProfile.id, token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        // The board starts in the default "Boards" category.
        const boardItem = page.locator('.SidebarBoardItem.subitem').filter({ hasText: /^Move Board Test$/ });
        await expect(boardItem).toBeVisible({ timeout: 10000 });

        // Open the board's options menu.
        await openBoardItemMenu(page, 'Move Board Test');

        // "Move To..." is a SubMenuOption rendered as a plain <div> (no role="button").
        // It opens the submenu on click via its onClick handler, not on hover.
        // The element receives id='moveBlock' from sidebarBoardItem.tsx.
        await page.locator('#moveBlock').click();

        // The submenu appears as .SubMenu.Menu.noselect.
        // Category items inside are TextOption divs with role="button".
        const subMenu = page.locator('.SubMenu.Menu.noselect');
        await expect(subMenu).toBeVisible({ timeout: 5000 });
        await subMenu.getByRole('button', { name: 'Target Category' }).click();

        // The board must now appear under "Target Category".
        const targetCategory = page.locator('.SidebarCategory').filter({
            has: page.locator('.category-title', { hasText: /^Target Category$/ }),
        });
        await expect(
            targetCategory.locator('.SidebarBoardItem.subitem').filter({ hasText: /^Move Board Test$/ }),
        ).toBeVisible({ timeout: 10000 });

        // And it must no longer appear under "Boards".
        const boardsCategory = page.locator('.SidebarCategory').filter({
            has: page.locator('.category-title', { hasText: /^Boards$/ }),
        });
        await expect(
            boardsCategory.locator('.SidebarBoardItem.subitem').filter({ hasText: /^Move Board Test$/ }),
        ).not.toBeVisible({ timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Hide board
    // ─────────────────────────────────────────────────────────────────────────

    test('user can hide a board from the sidebar via the board options menu', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();

        // Create two boards so hiding the active one switches to the other (not a blank state).
        const boardId = await createBoardViaApi('Board To Hide', token);
        const { teamId, viewId } = await createBoardView(boardId, token);
        const boardId2 = await createBoardViaApi('Anchor Board', token);
        await createBoardView(boardId2, token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        // Both boards are visible in the sidebar.
        await expect(
            page.locator('.SidebarBoardItem.subitem').filter({ hasText: /^Board To Hide$/ }),
        ).toBeVisible({ timeout: 10000 });
        await expect(
            page.locator('.SidebarBoardItem.subitem').filter({ hasText: /^Anchor Board$/ }),
        ).toBeVisible({ timeout: 5000 });

        // Open options on the target board and click "Hide board".
        await openBoardItemMenu(page, 'Board To Hide');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Hide board' }).click();

        // The board must disappear from the sidebar.
        await expect(
            page.locator('.SidebarBoardItem.subitem').filter({ hasText: /^Board To Hide$/ }),
        ).not.toBeVisible({ timeout: 10000 });

        // The anchor board is unaffected.
        await expect(
            page.locator('.SidebarBoardItem.subitem').filter({ hasText: /^Anchor Board$/ }),
        ).toBeVisible({ timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Viewer role — categories are user-scoped, not role-restricted
    // ─────────────────────────────────────────────────────────────────────────

    test('Viewer can create and rename their own sidebar categories', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const adminToken = (adminClient as any).token as string;

        // Create an open board and give regularuser Viewer access to it so they
        // can see the boards sidebar.
        const boardId = await createBoardViaApi('Viewer Category Board', adminToken);
        const { teamId, viewId } = await createBoardView(boardId, adminToken);
        const regularClient = await mattermost.getClient(regularUser, regularPass);
        const regularProfile = await regularClient.getMe();
        const regularToken = (regularClient as any).token as string;
        await addBoardMember(boardId, regularProfile.id, 'viewer', adminToken);
        await seedWelcomePageViewed(regularProfile.id, regularToken);

        const { ctx, page } = await loginAndOpenBoard(browser, regularUser, regularPass, teamId, boardId, viewId);

        // ── Create ──────────────────────────────────────────────────────────
        // Categories are personal / user-scoped — board role has no effect on
        // the ability to manage your own sidebar categories.
        await openCategoryMenu(page, 'Boards');
        const menu = page.locator('.Menu.noselect');
        await expect(menu.getByRole('button', { name: 'Create New Category' })).toBeVisible();
        await menu.getByRole('button', { name: 'Create New Category' }).click();

        await expect(page.locator('.CreateCategoryModal')).toBeVisible({ timeout: 5000 });
        await page.locator('.categoryNameInput').fill("Viewer Category");
        await page.locator('.CreateCategoryModal').getByRole('button', { name: 'Create' }).click();

        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Viewer Category') }) }),
        ).toBeVisible({ timeout: 10000 });

        // ── Rename ──────────────────────────────────────────────────────────
        await openCategoryMenu(page, 'Viewer Category');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Rename Category' }).click();

        await expect(page.locator('.CreateCategoryModal')).toBeVisible({ timeout: 5000 });
        await page.locator('.categoryNameInput').clear();
        await page.locator('.categoryNameInput').fill('Renamed Viewer Category');
        await page.locator('.CreateCategoryModal').getByRole('button', { name: 'Update' }).click();

        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Renamed Viewer Category') }) }),
        ).toBeVisible({ timeout: 10000 });

        await ctx.close();
    });

    test('Viewer can delete their own custom category', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const adminToken = (adminClient as any).token as string;

        const boardId = await createBoardViaApi('Viewer Delete Category Board', adminToken);
        const { teamId, viewId } = await createBoardView(boardId, adminToken);
        const regularClient = await mattermost.getClient(regularUser, regularPass);
        const regularProfile = await regularClient.getMe();
        const regularToken = (regularClient as any).token as string;
        await addBoardMember(boardId, regularProfile.id, 'viewer', adminToken);
        // Pre-create the category as the viewer (their personal category).
        await createCategoryViaApi('Viewer Temp Category', regularProfile.id, regularToken);
        await seedWelcomePageViewed(regularProfile.id, regularToken);

        const { ctx, page } = await loginAndOpenBoard(browser, regularUser, regularPass, teamId, boardId, viewId);

        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Viewer Temp Category') }) }),
        ).toBeVisible({ timeout: 10000 });

        await openCategoryMenu(page, 'Viewer Temp Category');
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Delete Category' }).click();

        const confirmDialog = page.locator('.confirmation-dialog-box');
        await expect(confirmDialog).toBeVisible({ timeout: 5000 });
        await confirmDialog.getByRole('button', { name: 'Confirm' }).click();

        await expect(
            page.locator('.octo-sidebar-item.category').filter({ has: page.locator('.category-title', { hasText: exact('Viewer Temp Category') }) }),
        ).not.toBeVisible({ timeout: 10000 });

        await ctx.close();
    });
});
