import { test, expect, type Browser, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { Users } from 'helpers/users';
import {
    addBoardMember,
    createCardViaApi,
    createNamedCardViaUi,
    openBoard,
    seedBoard,
} from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

test.describe('Realtime Collaboration', () => {
    test.describe.configure({ timeout: 300000 });

    async function openSharedBoard(browser: Browser, title: string, cardTitles: string[] = []): Promise<{
        pageA: Page;
        pageB: Page;
        close: () => Promise<void>;
    }> {
        const board = await seedBoard(mattermost, Users.admin.username, Users.admin.password, title);
        for (const cardTitle of cardTitles) {
            await createCardViaApi(mattermost, board.boardId, cardTitle, board.token);
        }
        const memberClient = await mattermost.getClient(Users.regularUser.username, Users.regularUser.password);
        const member = await memberClient.getMe();
        await addBoardMember(mattermost, board.boardId, member.id, 'editor', board.token);

        const ctxA = await browser.newContext();
        const ctxB = await browser.newContext();
        const pageA = await ctxA.newPage();
        const pageB = await ctxB.newPage();

        await openBoard(pageA, mattermost, Users.admin.username, Users.admin.password, board);
        await openBoard(pageB, mattermost, Users.regularUser.username, Users.regularUser.password, board);

        return {
            pageA,
            pageB,
            close: async () => {
                await ctxA.close();
                await ctxB.close();
            },
        };
    }

    test('a card created by one user appears for another without a reload', async ({ browser }) => {
        const { pageA, pageB, close } = await openSharedBoard(browser, 'Realtime Card Board');

        await createNamedCardViaUi(pageA, 'Realtime Card');

        await expect(pageB.locator('.KanbanCard').filter({ hasText: 'Realtime Card' }))
            .toBeVisible({ timeout: 15000 });

        await close();
    });

    test('a board title change syncs to another user', async ({ browser }) => {
        const { pageA, pageB, close } = await openSharedBoard(browser, 'Realtime Title Board');

        const titleInput = pageA.locator('.ViewTitle .Editable');
        await titleInput.click();
        await titleInput.fill('Realtime Title Renamed');
        await titleInput.press('Enter');

        await expect(pageB.locator('.octo-sidebar-list')).toContainText('Realtime Title Renamed', { timeout: 15000 });
        await expect(pageB.locator('.ViewTitle .Editable')).toHaveValue('Realtime Title Renamed', { timeout: 15000 });

        await close();
    });

    test('a card title edit syncs to another user', async ({ browser }) => {
        const { pageA, pageB, close } = await openSharedBoard(browser, 'Realtime Edit Board', ['Before Edit']);
        await expect(pageB.locator('.KanbanCard').filter({ hasText: 'Before Edit' })).toBeVisible({ timeout: 15000 });
        await expect(pageA.locator('.KanbanCard').filter({ hasText: 'Before Edit' })).toBeVisible({ timeout: 15000 });

        await pageA.locator('.KanbanCard').filter({ hasText: 'Before Edit' }).click();
        const dialog = pageA.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });
        const titleField = dialog.locator('.CardDetail .Editable.title:not([disabled])');
        await titleField.click();
        await titleField.fill('After Edit');
        await dialog.locator('.dialog__close').click();

        await expect(pageB.locator('.KanbanCard').filter({ hasText: 'After Edit' })).toBeVisible({ timeout: 15000 });
        await expect(pageB.locator('.KanbanCard').filter({ hasText: 'Before Edit' })).toHaveCount(0);

        await close();
    });

    test('a card can be unfollowed and followed again from its dialog', async ({ browser }) => {
        const { pageA, close } = await openSharedBoard(browser, 'Follow Card Board', ['Followable Card']);

        const card = pageA.locator('.KanbanCard').filter({ hasText: 'Followable Card' });
        await expect(card).toBeVisible({ timeout: 15000 });
        await card.click();
        const dialog = pageA.locator('.Dialog.cardDialog');
        await expect(dialog).toBeVisible({ timeout: 15000 });

        // The card creator is subscribed automatically.
        await expect(dialog.locator('.cardFollowBtn.unfollow')).toContainText('Following', { timeout: 10000 });

        await dialog.locator('.cardFollowBtn.unfollow').click();
        await expect(dialog.locator('.cardFollowBtn.follow')).toBeVisible({ timeout: 10000 });

        await dialog.locator('.cardFollowBtn.follow').click();
        await expect(dialog.locator('.cardFollowBtn.unfollow')).toContainText('Following', { timeout: 10000 });

        await close();
    });
});
