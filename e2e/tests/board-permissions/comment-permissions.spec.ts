// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { createBoardViaApi, getBoardMeta, seedWelcomePageViewed, addBoardMember } from 'helpers/boards';

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
// API Helpers
// ─────────────────────────────────────────────────────────────────────────────

/** Create a card block on a board, returns the card's id. */
async function createCardViaApi(boardId: string, title: string, token: string): Promise<string> {
    const now = Date.now();
    const cardId = `card${boardId.substring(0, 8)}${now}`.replace(/[^a-z0-9]/gi, '').substring(0, 26);

    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/blocks`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify([{
            id: cardId,
            boardId,
            parentId: boardId,
            type: 'card',
            title,
            schema: 1,
            createAt: now,
            updateAt: now,
            fields: {
                icon: '',
                properties: {},
                contentOrder: [],
                isTemplate: false,
            },
        }]),
    });
    if (!resp.ok) {
        throw new Error(`Failed to create card: ${resp.status}`);
    }
    const data = (await resp.json()) as Array<{ id: string }>;
    return data[0]?.id ?? cardId;
}

/** Create a comment block on a card, returns the comment's id. */
async function createCommentViaApi(boardId: string, cardId: string, text: string, token: string): Promise<string> {
    const now = Date.now();
    const commentId = `cmt${cardId.substring(0, 8)}${now}`.replace(/[^a-z0-9]/gi, '').substring(0, 26);

    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/blocks`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify([{
            id: commentId,
            boardId,
            parentId: cardId,
            type: 'comment',
            title: text,
            schema: 1,
            createAt: now,
            updateAt: now,
            fields: {},
        }]),
    });
    if (!resp.ok) {
        throw new Error(`Failed to create comment: ${resp.status}`);
    }
    const data = (await resp.json()) as Array<{ id: string }>;
    return data[0]?.id ?? commentId;
}

/**
 * Log in as `username`, navigate directly to the card URL, and wait for the
 * card dialog to appear. Returns after the `.Dialog.cardDialog` is visible.
 */
async function openCardAsUser(
    browser: import('@playwright/test').Browser,
    username: string,
    password: string,
    teamId: string,
    boardId: string,
    viewId: string,
    cardId: string,
) {
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

    // Navigate directly to the card URL — the router renders the card dialog on top of the board.
    const cardUrl = `${mattermost.url()}/boards/team/${teamId}/${boardId}/${viewId}/${cardId}`;
    await page.goto(cardUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });

    await expect(page.locator('.Dialog.cardDialog')).toBeVisible({ timeout: 20000 });

    return { ctx, page };
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Comment Permissions', () => {
    test.describe.configure({ timeout: 300000 });

    // ─────────────────────────────────────────────────────────────────────────
    // Adding comments
    // ─────────────────────────────────────────────────────────────────────────

    test('Viewer cannot see the comment input', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;

        const boardId = await createBoardViaApi(mattermost, 'Viewer Comment Board', token);
        const { teamId, viewId } = await getBoardMeta(mattermost, boardId, token);
        const cardId = await createCardViaApi(boardId, 'Viewer Card', token);

        const viewerClient = await mattermost.getClient(regularUser, regularPass);
        const viewerProfile = await viewerClient.getMe();
        const viewerToken = (viewerClient as any).token as string;
        await addBoardMember(mattermost, boardId, viewerProfile.id, 'viewer', token);
        await seedWelcomePageViewed(mattermost, viewerProfile.id, viewerToken);

        const { ctx, page } = await openCardAsUser(browser, regularUser, regularPass, teamId, boardId, viewId, cardId);

        // Viewer has no CommentBoardCards permission — the input area is hidden.
        await expect(page.locator('.CommentsList__new')).not.toBeVisible({ timeout: 5000 });

        await ctx.close();
    });

    test('Commenter can add a comment', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;

        const boardId = await createBoardViaApi(mattermost, 'Commenter Add Comment Board', token);
        const { teamId, viewId } = await getBoardMeta(mattermost, boardId, token);
        const cardId = await createCardViaApi(boardId, 'Commenter Card', token);

        const commenterClient = await mattermost.getClient(regularUser, regularPass);
        const commenterProfile = await commenterClient.getMe();
        const commenterToken = (commenterClient as any).token as string;
        await addBoardMember(mattermost, boardId, commenterProfile.id, 'commenter', token);
        await seedWelcomePageViewed(mattermost, commenterProfile.id, commenterToken);

        const { ctx, page } = await openCardAsUser(browser, regularUser, regularPass, teamId, boardId, viewId, cardId);

        // Commenter has CommentBoardCards permission — the input area is visible.
        await expect(page.locator('.CommentsList__new')).toBeVisible({ timeout: 10000 });

        // Click the preview element to enter edit mode, then type the comment.
        await page.locator('.CommentsList__new [data-testid="preview-element"]').click();
        await page.locator('.CommentsList__new [contenteditable="true"]').waitFor({ state: 'visible', timeout: 5000 });
        await page.locator('.CommentsList__new [contenteditable="true"]').fill('Hello from commenter');

        // The Send button appears only when the comment text is non-empty.
        await expect(page.locator('.CommentsList__new button.Button.filled')).toBeVisible({ timeout: 5000 });
        await page.locator('.CommentsList__new button.Button.filled').click();

        // The new comment should appear in the list.
        await expect(page.locator('.CommentsList .Comment.comment .comment-markdown')).toContainText(
            'Hello from commenter',
            { timeout: 10000 },
        );

        await ctx.close();
    });

    test('Editor can add a comment', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;

        const boardId = await createBoardViaApi(mattermost, 'Editor Add Comment Board', token);
        const { teamId, viewId } = await getBoardMeta(mattermost, boardId, token);
        const cardId = await createCardViaApi(boardId, 'Editor Card', token);

        const editorClient = await mattermost.getClient(regularUser, regularPass);
        const editorProfile = await editorClient.getMe();
        const editorToken = (editorClient as any).token as string;
        await addBoardMember(mattermost, boardId, editorProfile.id, 'editor', token);
        await seedWelcomePageViewed(mattermost, editorProfile.id, editorToken);

        const { ctx, page } = await openCardAsUser(browser, regularUser, regularPass, teamId, boardId, viewId, cardId);

        await expect(page.locator('.CommentsList__new')).toBeVisible({ timeout: 10000 });

        await page.locator('.CommentsList__new [data-testid="preview-element"]').click();
        await page.locator('.CommentsList__new [contenteditable="true"]').waitFor({ state: 'visible', timeout: 5000 });
        await page.locator('.CommentsList__new [contenteditable="true"]').fill('Hello from editor');

        await expect(page.locator('.CommentsList__new button.Button.filled')).toBeVisible({ timeout: 5000 });
        await page.locator('.CommentsList__new button.Button.filled').click();

        await expect(page.locator('.CommentsList .Comment.comment .comment-markdown')).toContainText(
            'Hello from editor',
            { timeout: 10000 },
        );

        await ctx.close();
    });

    test('Admin can add a comment', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const token = (adminClient as any).token as string;

        const boardId = await createBoardViaApi(mattermost, 'Admin Add Comment Board', token);
        const { teamId, viewId } = await getBoardMeta(mattermost, boardId, token);
        const cardId = await createCardViaApi(boardId, 'Admin Card', token);

        // Admin is the board creator — no need to addBoardMember.
        // Seed welcomePageViewed for the admin user as well.
        const adminProfile = await adminClient.getMe();
        await seedWelcomePageViewed(mattermost, adminProfile.id, token);

        const { ctx, page } = await openCardAsUser(browser, adminUser, adminPass, teamId, boardId, viewId, cardId);

        await expect(page.locator('.CommentsList__new')).toBeVisible({ timeout: 10000 });

        await page.locator('.CommentsList__new [data-testid="preview-element"]').click();
        await page.locator('.CommentsList__new [contenteditable="true"]').waitFor({ state: 'visible', timeout: 5000 });
        await page.locator('.CommentsList__new [contenteditable="true"]').fill('Hello from admin');

        await expect(page.locator('.CommentsList__new button.Button.filled')).toBeVisible({ timeout: 5000 });
        await page.locator('.CommentsList__new button.Button.filled').click();

        await expect(page.locator('.CommentsList .Comment.comment .comment-markdown')).toContainText(
            'Hello from admin',
            { timeout: 10000 },
        );

        await ctx.close();
    });

    // ─────────────────────────────────────────────────────────────────────────
    // Deleting comments
    // ─────────────────────────────────────────────────────────────────────────

    test('Admin can delete another user\'s comment', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const adminToken = (adminClient as any).token as string;

        const boardId = await createBoardViaApi(mattermost, 'Admin Delete Comment Board', adminToken);
        const { teamId, viewId } = await getBoardMeta(mattermost, boardId, adminToken);
        const cardId = await createCardViaApi(boardId, 'Delete Test Card', adminToken);

        // Create a comment as regularuser so the admin can attempt to delete it.
        const regularClient = await mattermost.getClient(regularUser, regularPass);
        const regularProfile = await regularClient.getMe();
        const regularToken = (regularClient as any).token as string;
        await addBoardMember(mattermost, boardId, regularProfile.id, 'commenter', adminToken);
        await createCommentViaApi(boardId, cardId, 'Commenter comment to delete', regularToken);

        // Seed welcomePageViewed for admin.
        const adminProfile = await adminClient.getMe();
        await seedWelcomePageViewed(mattermost, adminProfile.id, adminToken);

        const { ctx, page } = await openCardAsUser(browser, adminUser, adminPass, teamId, boardId, viewId, cardId);

        // The comment should be visible.
        const comment = page.locator('.CommentsList .Comment.comment').filter({ hasText: 'Commenter comment to delete' });
        await expect(comment).toBeVisible({ timeout: 10000 });

        // Admin has DeleteOthersComments permission — hover to reveal the options menu
        // (CSS hides .MenuWrapper by default and shows it only on .Comment:hover).
        await comment.hover();

        // Click the options icon directly with force:true since the MenuWrapper is
        // revealed only via CSS :hover and Playwright's visibility check races with
        // the CSS state change in headless mode.
        await comment.locator('.comment-header .MenuWrapper button.IconButton').click({ force: true });
        await expect(page.locator('.Menu.noselect')).toBeVisible({ timeout: 5000 });

        // The Delete item should be present and clickable.
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Delete' }).click();

        // Comment should disappear after deletion.
        await expect(comment).not.toBeVisible({ timeout: 10000 });

        await ctx.close();
    });

    test('Commenter sees delete option on own comment but not on another user\'s comment', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const adminToken = (adminClient as any).token as string;

        const boardId = await createBoardViaApi(mattermost, 'Commenter Delete Board', adminToken);
        const { teamId, viewId } = await getBoardMeta(mattermost, boardId, adminToken);
        const cardId = await createCardViaApi(boardId, 'Commenter Delete Card', adminToken);

        // Create a comment as admin (the "other" user's comment).
        await createCommentViaApi(boardId, cardId, 'Admin comment (not deletable by commenter)', adminToken);

        const regularClient = await mattermost.getClient(regularUser, regularPass);
        const regularProfile = await regularClient.getMe();
        const regularToken = (regularClient as any).token as string;
        await addBoardMember(mattermost, boardId, regularProfile.id, 'commenter', adminToken);
        await seedWelcomePageViewed(mattermost, regularProfile.id, regularToken);

        // Create a comment as the commenter (their own comment).
        const ownCommentId = await createCommentViaApi(boardId, cardId, 'Commenter own comment', regularToken);

        const { ctx, page } = await openCardAsUser(browser, regularUser, regularPass, teamId, boardId, viewId, cardId);

        // Both comments must be visible.
        const ownComment = page.locator('.CommentsList .Comment.comment').filter({ hasText: 'Commenter own comment' });
        const othersComment = page.locator('.CommentsList .Comment.comment').filter({ hasText: 'Admin comment' });
        await expect(ownComment).toBeVisible({ timeout: 10000 });
        await expect(othersComment).toBeVisible({ timeout: 5000 });

        // Commenter role only has CommentBoardCards permission, not delete. Verify the server
        // rejects the DELETE request with 403 even for their own comment.
        const deleteResp = await fetch(
            `${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/blocks/${ownCommentId}`,
            {
                method: 'DELETE',
                headers: {
                    Authorization: `Bearer ${regularToken}`,
                    'X-Requested-With': 'XMLHttpRequest',
                },
            },
        );
        expect(deleteResp.status).toBe(403);

        // The comment must still be visible — no optimistic removal on 403.
        await expect(ownComment).toBeVisible({ timeout: 5000 });

        // Commenter CANNOT see the delete option on another user's comment — the MenuWrapper is not
        // rendered at all when readonly=true in comment.tsx.
        await expect(othersComment.locator('.comment-header .MenuWrapper')).not.toBeAttached();

        await ctx.close();
    });

    test('Viewer cannot delete any comment', async ({ browser }) => {
        const adminClient = await mattermost.getAdminClient();
        const adminToken = (adminClient as any).token as string;

        const boardId = await createBoardViaApi(mattermost, 'Viewer Delete Board', adminToken);
        const { teamId, viewId } = await getBoardMeta(mattermost, boardId, adminToken);
        const cardId = await createCardViaApi(boardId, 'Viewer Delete Card', adminToken);

        // Create a comment as admin.
        await createCommentViaApi(boardId, cardId, 'Admin comment visible to viewer', adminToken);

        const viewerClient = await mattermost.getClient(regularUser, regularPass);
        const viewerProfile = await viewerClient.getMe();
        const viewerToken = (viewerClient as any).token as string;
        await addBoardMember(mattermost, boardId, viewerProfile.id, 'viewer', adminToken);
        await seedWelcomePageViewed(mattermost, viewerProfile.id, viewerToken);

        const { ctx, page } = await openCardAsUser(browser, regularUser, regularPass, teamId, boardId, viewId, cardId);

        const comment = page.locator('.CommentsList .Comment.comment').filter({ hasText: 'Admin comment visible to viewer' });
        await expect(comment).toBeVisible({ timeout: 10000 });

        // Viewer has no delete permission — the options MenuWrapper must not be present.
        await expect(comment.locator('.comment-header .MenuWrapper')).not.toBeVisible({ timeout: 5000 });

        await ctx.close();
    });
});
