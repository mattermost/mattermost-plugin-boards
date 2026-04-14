import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';

const adminUser = 'admin';
const adminPass = 'admin';
const regularUser = 'regularuser';
const regularPass = 'regularuser';
const secondUser = 'seconduser';
const secondPass = 'seconduser';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Log in and navigate to boards. Creates a blank board with the given title if
 * the user has none yet.
 */
async function setupBoard(page: Page, username: string, password: string, boardTitle: string): Promise<void> {
    const mmPage = new MattermostPage(page);
    await mmPage.login(mattermost.url(), username, password);
    await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
    await mmPage.navigateToBoardsFromUrl(mattermost.url());

    const boardComponent = page.locator('.BoardComponent');
    const boardLoaded = await boardComponent
        .waitFor({ state: 'visible', timeout: 5000 })
        .then(() => true)
        .catch(() => false);

    if (!boardLoaded) {
        await expect(page.locator('.BoardTemplateSelector')).toBeVisible({ timeout: 15000 });
        await page.locator('.templates-sidebar__footer button').click();
        await expect(boardComponent).toBeVisible({ timeout: 15000 });

        const titleInput = page.locator('.ViewTitle .Editable');
        await titleInput.click();
        await titleInput.fill(boardTitle);
        await titleInput.press('Enter');
        await expect(page.locator('.octo-sidebar-list')).toContainText(boardTitle, { timeout: 10000 });
    }

    await expect(boardComponent).toBeVisible({ timeout: 15000 });
}

/** Open the Share dialog for the currently active board. */
async function openShareDialog(page: Page): Promise<void> {
    await page.locator('.ShareBoardButton button').click();
    await expect(page.locator('.ShareBoardDialog')).toBeVisible({ timeout: 10000 });
}

/** Create a board via the Focalboard REST API, returns the new board's id. */
async function createBoardViaApi(
    title: string,
    type: 'O' | 'P',
    token: string,
): Promise<string> {
    const adminClient = await mattermost.getAdminClient();
    const teams = await adminClient.getMyTeams();
    const teamId = teams[0]?.id;
    if (!teamId) {
        throw new Error('No team found');
    }

    const resp = await fetch(
        `${mattermost.url()}/plugins/focalboard/api/v2/boards`,
        {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            body: JSON.stringify({ teamId, title, type, cardProperties: [] }),
        },
    );
    if (!resp.ok) {
        throw new Error(`Failed to create board: ${resp.status}`);
    }
    const data = (await resp.json()) as { id: string };
    return data.id;
}

/** Return (teamId, defaultViewId) for a board — needed for the full board URL.
 *  Views are blocks of type "view"; there is no /views endpoint.
 *  If the board has no views (newly created via API), one is created first. */
async function getBoardMeta(boardId: string, token: string): Promise<{ teamId: string; viewId: string }> {
    const adminClient = await mattermost.getAdminClient();
    const teams = await adminClient.getMyTeams();
    const teamId = teams[0]?.id ?? '';

    const headers = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
    };

    // Boards created via API have no views — create a default board view.
    // The server requires id, createAt, and updateAt fields.
    const now = Date.now();
    const viewBlockId = `view-${boardId.substring(0, 8)}-${now}`.replace(/[^a-z0-9]/gi, '').substring(0, 26);
    const createResp = await fetch(
        `${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/blocks`,
        {
            method: 'POST',
            headers,
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
        },
    );
    if (!createResp.ok) {
        throw new Error(`Failed to create board view: ${createResp.status}`);
    }
    const created = (await createResp.json()) as Array<{ id: string }>;
    const viewId = created[0]?.id ?? '';
    return { teamId, viewId };
}

/**
 * Pre-seed the Focalboard welcomePageViewed config for a user so that the
 * welcome page never appears on first boards visit. Without this, FBRoute
 * redirects to /welcome on the first load (after fetchMe sets loggedIn=true
 * with empty myConfig), and the welcome page's goForward() call during a
 * render cycle triggers a React rendering error caught by ErrorBoundary.
 */
async function seedWelcomePageViewed(userId: string, userToken: string): Promise<void> {
    const resp = await fetch(
        `${mattermost.url()}/plugins/focalboard/api/v2/users/${userId}/config`,
        {
            method: 'PUT',
            headers: {
                Authorization: `Bearer ${userToken}`,
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            body: JSON.stringify({ updatedFields: { welcomePageViewed: '1' } }),
        },
    );
    if (!resp.ok) {
        throw new Error(`Failed to seed welcomePageViewed: ${resp.status}`);
    }
}

/** Add a user to a board with the given scheme flags. */
async function addBoardMember(
    boardId: string,
    userId: string,
    role: 'admin' | 'editor' | 'commenter' | 'viewer',
    token: string,
): Promise<void> {
    const body = {
        boardId,
        userId,
        schemeAdmin: role === 'admin',
        schemeEditor: role === 'editor',
        schemeCommenter: role === 'commenter',
        schemeViewer: role === 'viewer',
    };
    const resp = await fetch(
        `${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/members`,
        {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            body: JSON.stringify(body),
        },
    );
    if (!resp.ok) {
        throw new Error(`Failed to add board member: ${resp.status}`);
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Board Member Roles', () => {
    test.describe.configure({ timeout: 300000 });

    // ─────────────────────────────────────────────────────────────────────────
    // Share dialog basics
    // ─────────────────────────────────────────────────────────────────────────

    test('share dialog opens and shows the board creator as admin', async ({ page }) => {
        await setupBoard(page, adminUser, adminPass, 'Admin Board');

        await openShareDialog(page);
        const shareDialog = page.locator('.ShareBoardDialog');

        // The creator's row should show "Admin" role.
        // Note: the team row ("Everyone at … Team") comes first in the list, so
        // we must filter by username rather than using .first().
        const memberList = shareDialog.locator('.user-items');
        await expect(memberList).toBeVisible({ timeout: 5000 });
        const creatorRow = memberList.locator('.user-item').filter({ hasText: adminUser });
        await expect(creatorRow.locator('.user-item__button')).toContainText('Admin', { timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Add a member via UI
    // ─────────────────────────────────────────────────────────────────────────

    test('board admin can add another user as a member', async ({ page }) => {
        await setupBoard(page, adminUser, adminPass, 'Sharing Board');

        await openShareDialog(page);
        const shareDialog = page.locator('.ShareBoardDialog');

        // Type into the user-search react-select and pick the suggestion.
        // Must use pressSequentially — fill() doesn't fire onInputChange,
        // so loadOptions is never called and the dropdown stays empty.
        const searchInput = shareDialog.locator('.userSearchInput input');
        await searchInput.click();
        await searchInput.pressSequentially(regularUser, { delay: 50 });

        // The userSearchInput Select has no classNamePrefix, so option elements
        // don't get a stable CSS class. The first match is auto-focused by
        // react-select. Wait for the @username text to confirm options loaded,
        // then press Enter to select the focused option.
        await expect(page.locator('strong').filter({ hasText: `@${regularUser}` })).toBeVisible({ timeout: 10000 });
        await searchInput.press('Enter');

        // The new member should appear in the list.
        await expect(
            shareDialog.locator('.user-items .user-item').filter({ hasText: regularUser }),
        ).toBeVisible({ timeout: 10000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Change a member's role
    // ─────────────────────────────────────────────────────────────────────────

    test('board admin can change a member role to Viewer', async ({ page }) => {
        await setupBoard(page, adminUser, adminPass, 'Role Change Board');

        await openShareDialog(page);
        const shareDialog = page.locator('.ShareBoardDialog');

        // Add regularuser via the UI search.
        const searchInput = shareDialog.locator('.userSearchInput input');
        await searchInput.click();
        await searchInput.pressSequentially(regularUser, { delay: 50 });
        await expect(page.locator('strong').filter({ hasText: `@${regularUser}` })).toBeVisible({ timeout: 10000 });
        await searchInput.press('Enter');

        const memberRow = shareDialog.locator('.user-items .user-item').filter({ hasText: regularUser });
        await expect(memberRow).toBeVisible({ timeout: 10000 });

        // Open the role dropdown and pick "Viewer".
        await memberRow.locator('.user-item__button').click();
        const roleMenu = page.locator('.Menu.noselect');
        await expect(roleMenu).toBeVisible({ timeout: 5000 });
        await roleMenu.getByRole('button', { name: 'Viewer' }).click();

        // The row should now reflect "Viewer".
        await expect(memberRow.locator('.user-item__button')).toContainText('Viewer', { timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Remove a member
    // ─────────────────────────────────────────────────────────────────────────

    test('board admin can remove a member from the board', async ({ page }) => {
        await setupBoard(page, adminUser, adminPass, 'Remove Member Board');

        await openShareDialog(page);
        const shareDialog = page.locator('.ShareBoardDialog');

        // Add seconduser.
        const searchInput = shareDialog.locator('.userSearchInput input');
        await searchInput.click();
        await searchInput.pressSequentially(secondUser, { delay: 50 });
        await expect(page.locator('strong').filter({ hasText: `@${secondUser}` })).toBeVisible({ timeout: 10000 });
        await searchInput.press('Enter');

        const memberRow = shareDialog.locator('.user-items .user-item').filter({ hasText: secondUser });
        await expect(memberRow).toBeVisible({ timeout: 10000 });

        // Open role dropdown → remove member.
        await memberRow.locator('.user-item__button').click();
        const roleMenu = page.locator('.Menu.noselect');
        await expect(roleMenu).toBeVisible({ timeout: 5000 });
        await roleMenu.getByRole('button', { name: 'Remove member' }).click();

        // Row should disappear.
        await expect(memberRow).not.toBeVisible({ timeout: 10000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Team-level role: make board private
    // ─────────────────────────────────────────────────────────────────────────

    test('setting team role to None makes the board private', async ({ page }) => {
        await setupBoard(page, adminUser, adminPass, 'Private Test Board');

        await openShareDialog(page);
        const shareDialog = page.locator('.ShareBoardDialog');

        // Find the "Everyone at … Team" row.
        const teamRow = shareDialog.locator('.user-item').filter({ hasText: 'Everyone at' });
        await expect(teamRow).toBeVisible({ timeout: 5000 });

        // Open dropdown and select "None".
        await teamRow.locator('.user-item__button').click();
        const roleMenu = page.locator('.Menu.noselect');
        await expect(roleMenu).toBeVisible({ timeout: 5000 });
        await roleMenu.getByRole('button', { name: 'None' }).click();

        // Team row role button should now say "None".
        await expect(teamRow.locator('.user-item__button')).toContainText('None', { timeout: 5000 });
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Private board: not visible to non-members
    // ─────────────────────────────────────────────────────────────────────────

    test('private board is not visible to non-members in the sidebar', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;

        // Create a private board (type 'P') via API.
        const boardId = await createBoardViaApi('Hidden Private Board', 'P', token);
        expect(boardId).toBeTruthy();

        // Log in as seconduser and verify the board does NOT appear in the sidebar.
        const ctx = await browser.newContext();
        const secondPage = await ctx.newPage();
        const mmPage = new MattermostPage(secondPage);
        await mmPage.login(mattermost.url(), secondUser, secondPass);
        await secondPage.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
        await mmPage.navigateToBoardsFromUrl(mattermost.url());

        // Wait for the sidebar to finish loading.
        await secondPage.locator('.Sidebar.octo-sidebar').waitFor({ state: 'visible', timeout: 15000 }).catch(() => {});

        await expect(secondPage.locator('.octo-sidebar-list')).not.toContainText('Hidden Private Board', { timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Viewer role: cannot create new cards
    // ─────────────────────────────────────────────────────────────────────────

    test('member with Viewer role cannot see the New card button', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;

        // Create an open board so seconduser can access it.
        const boardId = await createBoardViaApi('Viewer Role Board', 'O', token);
        expect(boardId).toBeTruthy();

        // Look up seconduser's MM id so we can add them as Viewer.
        const secondClient = await mattermost.getClient(secondUser, secondPass);
        const secondProfile = await secondClient.getMe();
        const secondToken = (secondClient as any).token as string;
        await addBoardMember(boardId, secondProfile.id, 'viewer', token);

        // Create a default view and get team/view IDs for the direct board URL.
        const { teamId, viewId } = await getBoardMeta(boardId, token);

        // Pre-seed welcomePageViewed so FBRoute never redirects to /welcome.
        // Without this, the welcome page's goForward() call during a render cycle
        // (triggered by patchProps after skip) mounts BoardPage mid-render and
        // causes a React rendering error caught by the ErrorBoundary.
        await seedWelcomePageViewed(secondProfile.id, secondToken);

        const ctx = await browser.newContext();
        const secondPage = await ctx.newPage();

        // Set lastTeamId before any page scripts run so HomeToCurrentTeam always
        // resolves a team and never redirects to error?id=unknown.
        await secondPage.addInitScript((tid) => {
            localStorage.setItem('lastTeamId', tid);
        }, teamId);

        const mmPage = new MattermostPage(secondPage);
        await mmPage.login(mattermost.url(), secondUser, secondPass);
        await secondPage.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });

        // Navigate directly to the full board URL, bypassing HomeToCurrentTeam.
        const boardUrl = `${mattermost.url()}/boards/team/${teamId}/${boardId}/${viewId}`;
        await secondPage.goto(boardUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });

        await expect(secondPage.locator('.BoardComponent')).toBeVisible({ timeout: 20000 });

        // Viewer should NOT see the "New" button.
        await expect(
            secondPage.locator('.ViewHeader .ButtonWithMenu .button-text'),
        ).not.toBeVisible({ timeout: 5000 });

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Editor role: can create new cards
    // ─────────────────────────────────────────────────────────────────────────

    test('member with Editor role can create new cards', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;

        const boardId = await createBoardViaApi('Editor Role Board', 'O', token);
        expect(boardId).toBeTruthy();

        const secondClient = await mattermost.getClient(secondUser, secondPass);
        const secondProfile = await secondClient.getMe();
        const secondToken = (secondClient as any).token as string;
        await addBoardMember(boardId, secondProfile.id, 'editor', token);

        const { teamId, viewId } = await getBoardMeta(boardId, token);

        // Pre-seed welcomePageViewed to prevent the welcome page render-cycle error.
        await seedWelcomePageViewed(secondProfile.id, secondToken);

        const ctx = await browser.newContext();
        const secondPage = await ctx.newPage();

        // Set lastTeamId before any page scripts run.
        await secondPage.addInitScript((tid) => {
            localStorage.setItem('lastTeamId', tid);
        }, teamId);

        const mmPage = new MattermostPage(secondPage);
        await mmPage.login(mattermost.url(), secondUser, secondPass);
        await secondPage.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });

        // Navigate directly to the full board URL, bypassing HomeToCurrentTeam.
        const boardUrl = `${mattermost.url()}/boards/team/${teamId}/${boardId}/${viewId}`;
        await secondPage.goto(boardUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });

        await expect(secondPage.locator('.BoardComponent')).toBeVisible({ timeout: 20000 });

        // Editor SHOULD see the "New" button.
        await expect(
            secondPage.locator('.ViewHeader .ButtonWithMenu .button-text'),
        ).toBeVisible({ timeout: 5000 });

        // Clicking "New" should open the card dialog.
        await secondPage.locator('.ViewHeader .ButtonWithMenu .button-text').click();
        await expect(secondPage.locator('.Dialog.cardDialog')).toBeVisible({ timeout: 10000 });

        await ctx.close();
    });
});
