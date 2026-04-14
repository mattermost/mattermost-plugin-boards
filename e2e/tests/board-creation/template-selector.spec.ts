// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import { test, expect, type Browser, type Page } from '@playwright/test';

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

/** Create a default board view; returns viewId. */
async function createBoardView(boardId: string, token: string): Promise<string> {
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
    return created[0]?.id ?? '';
}

/**
 * Create a board template via the Focalboard boards API with isTemplate=true.
 * Templates belong to the team and appear in the template selector for all team members.
 */
async function createTemplateViaApi(title: string, token: string): Promise<string> {
    const teamId = await getTeamId();
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({ teamId, title, type: 'O', cardProperties: [], isTemplate: true }),
    });
    if (!resp.ok) throw new Error(`Failed to create template: ${resp.status}`);
    const templateId = ((await resp.json()) as { id: string }).id;

    // A template with at least one view block is required.
    // When "Use this template" calls mutator.duplicateBoard(), the resulting board
    // inherits all blocks from the template. If there are no view blocks,
    // teamToBoardAndViewRedirect's `boardViews.length > 0` guard is never satisfied,
    // setCurrentView is never dispatched, and workspace.tsx renders null instead of
    // <CenterPanel> — so .BoardComponent never appears.
    await createBoardView(templateId, token);

    return templateId;
}

/**
 * Pre-seed welcomePageViewed so FBRoute never redirects to /welcome.
 * Without this the welcome page's goForward() during a render cycle triggers a
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
 * Navigate to /boards/team/{teamId} (no boardId in the URL) so the template
 * selector renders as the center content.
 *
 * Without the sessionStorage guard, teamToBoardAndViewRedirect immediately
 * redirects to UserSettings.lastBoardId[teamId] (localStorage) or the first
 * category board whenever match.params.boardId is absent — so the template
 * selector never mounts in sessions where the user has already visited boards.
 *
 * Setting sessionStorage key 'skip_board_redirect' = 'true' makes the component
 * skip that redirect (Constants.sessionStorageSkipBoardRedirectKey).  The flag is
 * automatically removed by the component's useEffect the moment a boardId appears
 * in the URL (e.g., after clicking "Use this template"), so it only applies to
 * this single navigation.
 */
async function openTemplateSelectorViaTeamUrl(page: Page, teamId: string): Promise<void> {
    // page.evaluate() runs in the current tab's sessionStorage (same origin as
    // the boards page), so the flag is available when the boards app initialises.
    await page.evaluate(() => {
        sessionStorage.setItem('skip_board_redirect', 'true');
    });

    await page.goto(
        `${mattermost.url()}/boards/team/${teamId}`,
        { waitUntil: 'domcontentloaded', timeout: 60000 },
    );
    await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
}

/**
 * Log in as `username`, navigate to a specific board URL so the sidebar is
 * fully loaded, and return the page/context.
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

    await page.goto(
        `${mattermost.url()}/boards/team/${teamId}/${boardId}/${viewId}`,
        { waitUntil: 'domcontentloaded', timeout: 60000 },
    );
    await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 20000 });

    return { ctx, page };
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Template Selector', () => {
    test.describe.configure({ timeout: 300000 });

    // ─────────────────────────────────────────────────────────────────────────
    // Template list basics
    // ─────────────────────────────────────────────────────────────────────────

    test('template list is populated with global templates', async ({ page }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await seedWelcomePageViewed(adminProfile.id, token);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        // Global templates are fetched from /api/v2/global/templates and rendered
        // as .BoardTemplateSelectorItem rows in the .templates-list sidebar.
        await expect(page.locator('.BoardTemplateSelectorItem').first()).toBeVisible({ timeout: 10000 });
        const count = await page.locator('.BoardTemplateSelectorItem').count();
        expect(count).toBeGreaterThan(0);
    });

    test('first template is selected by default', async ({ page }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await seedWelcomePageViewed(adminProfile.id, token);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        // activeTemplate state is initialised to allTemplates[0], which renders
        // with the `active` CSS class on the corresponding .BoardTemplateSelectorItem.
        await expect(page.locator('.BoardTemplateSelectorItem.active').first()).toBeVisible({ timeout: 10000 });
    });

    test('clicking a different template makes it active', async ({ page }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await seedWelcomePageViewed(adminProfile.id, token);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        const items = page.locator('.BoardTemplateSelectorItem');
        await expect(items.first()).toBeVisible({ timeout: 10000 });

        // Click the second template (index 1) to change the selection.
        const second = items.nth(1);
        await second.click();

        // The clicked item must receive the `active` class.
        await expect(second).toHaveClass(/active/, { timeout: 5000 });

        // The previously-first item must no longer be active.
        await expect(items.first()).not.toHaveClass(/active/, { timeout: 5000 });

        // "Use this template" button is always visible in the .buttons area.
        await expect(
            page.locator('.buttons').getByRole('button', { name: 'Use this template' }),
        ).toBeVisible({ timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Using a template
    // ─────────────────────────────────────────────────────────────────────────

    test('"Use this template" creates a board and opens it in the editor', async ({ page }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await seedWelcomePageViewed(adminProfile.id, token);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        // First template is active by default — click "Use this template" immediately.
        await expect(page.locator('.BoardTemplateSelectorItem').first()).toBeVisible({ timeout: 10000 });
        await page.locator('.buttons').getByRole('button', { name: 'Use this template' }).click();

        // mutator.addBoardFromTemplate() calls POST /boards/{id}/duplicate and
        // navigates to the new board — the board editor must appear.
        await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 20000 });

        // The new board also appears in the sidebar.
        await expect(
            page.locator('.octo-sidebar-list .octo-sidebar-item').first(),
        ).toBeVisible({ timeout: 10000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Opening the template selector from the sidebar (boards already exist)
    // ─────────────────────────────────────────────────────────────────────────

    test('template selector opens via the sidebar add-board button', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        const boardId = await createBoardViaApi('Sidebar Trigger Board', token);
        const viewId = await createBoardView(boardId, token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const { ctx, page } = await loginAndOpenBoard(browser, adminUser, adminPass, teamId, boardId, viewId);

        // The sidebar is rendered when a board is open.
        // Click the '+' IconButton (.add-board-icon) in the BoardsSwitcher to open
        // the "Add board" dropdown.
        await page.locator('.add-board-icon').click();

        // A small Menu appears — click the "Create new board" TextOption.
        const menu = page.locator('.Menu.noselect');
        await expect(menu).toBeVisible({ timeout: 5000 });
        await menu.getByRole('button', { name: 'Create new board' }).click();

        // The BoardTemplateSelector modal overlays the board editor.
        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 10000 });

        // Dismiss with the X IconButton in the .toolbar.
        await page.locator('.BoardTemplateSelector .toolbar .IconButton').click();
        await expect(page.locator('.BoardTemplateSelector')).not.toBeVisible({ timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Custom templates
    // ─────────────────────────────────────────────────────────────────────────

    test('custom template created via API appears in the template list', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await createTemplateViaApi('E2E Custom Template', token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const ctx = await browser.newContext();
        const page = await ctx.newPage();
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        // Custom templates (isTemplate: true, owned by the team) appear below the
        // global templates in the .templates-list sidebar.
        await expect(
            page.locator('.BoardTemplateSelectorItem').filter({ hasText: 'E2E Custom Template' }),
        ).toBeVisible({ timeout: 10000 });

        await ctx.close();
    });

    test('custom template can be used to create a board', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await createTemplateViaApi('E2E Usable Template', token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const ctx = await browser.newContext();
        const page = await ctx.newPage();
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        // Select the custom template.
        const templateItem = page.locator('.BoardTemplateSelectorItem').filter({ hasText: 'E2E Usable Template' });
        await expect(templateItem).toBeVisible({ timeout: 10000 });
        await templateItem.click();
        await expect(templateItem).toHaveClass(/active/, { timeout: 5000 });

        // Use it to create a board — mutator.addBoardFromTemplate() navigates to the new board.
        await page.locator('.buttons').getByRole('button', { name: 'Use this template' }).click();
        await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 20000 });

        await ctx.close();
    });

    test('custom template can be deleted via the template selector', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await createTemplateViaApi('E2E Deletable Template', token);
        await seedWelcomePageViewed(adminProfile.id, token);

        const ctx = await browser.newContext();
        const page = await ctx.newPage();
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        const templateItem = page.locator('.BoardTemplateSelectorItem').filter({ hasText: 'E2E Deletable Template' });
        await expect(templateItem).toBeVisible({ timeout: 10000 });

        // .actions is display:none by default and only revealed on :hover
        // (boardTemplateSelectorItem.scss line 33-35).
        await templateItem.hover();

        // The delete IconButton has title="Delete" (the first button in .actions).
        // System templates (createdBy === SystemUserID) never render .actions, so
        // this only works for user-created templates.
        await templateItem.locator('.actions button[title="Delete"]').click();

        // DeleteBoardDialog opens with the template-specific confirmation title.
        const dialog = page.locator('.DeleteBoardDialog');
        await expect(dialog).toBeVisible({ timeout: 5000 });
        await expect(dialog.locator('h2')).toContainText('Confirm delete board template');
        await dialog.getByRole('button', { name: 'Delete' }).click();

        // The template row must disappear from the list after deletion.
        await expect(templateItem).not.toBeVisible({ timeout: 10000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Create new template
    // ─────────────────────────────────────────────────────────────────────────

    test('"Create new template" opens a blank template board for editing', async ({ page }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;
        const adminProfile = await adminClient.getMe();
        const teamId = await getTeamId();
        await seedWelcomePageViewed(adminProfile.id, token);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), adminUser, adminPass);
        await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await openTemplateSelectorViaTeamUrl(page, teamId);

        // .new-template is the link-style Button at the top of the .templates-list
        // (calls mutator.addEmptyBoardTemplate() → creates a blank template and navigates to it).
        await page.locator('.new-template').click();

        // The board editor opens for the newly created template.
        await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 20000 });

        // workspace.tsx renders a .banner when board.isTemplate is true, distinguishing
        // template editing from regular board editing.
        await expect(
            page.locator('.banner').filter({ hasText: "You're editing a board template." }),
        ).toBeVisible({ timeout: 10000 });
    });
});
