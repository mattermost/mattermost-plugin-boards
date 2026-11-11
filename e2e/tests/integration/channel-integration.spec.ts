import { test, expect } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { Users } from 'helpers/users';
import {
    createCardViaApi,
    getTeamId,
    linkBoardToChannelViaApi,
    listTeamBoardsViaApi,
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

test.describe('Channel Integration', () => {
    test.describe.configure({ timeout: 300000 });

    const { username, password } = Users.admin;

    async function createChannel(name: string, displayName: string, memberUsernames: string[] = []) {
        const adminClient = await mattermost.getAdminClient();
        const teamId = await getTeamId(mattermost);
        const channel = await adminClient.createChannel({
            team_id: teamId,
            name,
            display_name: displayName,
            type: 'O',
        } as any);
        for (const member of memberUsernames) {
            const memberClient = await mattermost.getClient(member, member);
            const me = await memberClient.getMe();
            await adminClient.addToChannel(me.id, channel.id);
        }
        return { channel, teamId, adminClient };
    }

    test('a card link posted in a channel renders a board unfurl', async ({ page }) => {
        const board = await seedBoard(mattermost, username, password, 'Unfurl Board');
        const cardId = await createCardViaApi(mattermost, board.boardId, 'Unfurl Card', board.token);

        const adminClient = await mattermost.getAdminClient();
        const channel = await adminClient.getChannelByName(board.teamId, 'town-square');
        const link = `${mattermost.url()}/boards/team/${board.teamId}/${board.boardId}/${board.viewId}/${cardId}`;
        await adminClient.createPost({ channel_id: channel.id, message: link } as any);

        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), username, password);
        await page.goto(`${mattermost.url()}/test/channels/town-square`, { waitUntil: 'domcontentloaded' });

        const unfurl = page.locator('.FocalboardUnfurl').last();
        await expect(unfurl).toBeVisible({ timeout: 30000 });
        await expect(unfurl.locator('.card_title')).toContainText('Unfurl Card');
        await expect(unfurl.locator('.board_title')).toContainText('Unfurl Board');
    });

    test('creating a channel with a board links an empty board to it', async ({ page }) => {
        const mmPage = new MattermostPage(page);
        await mmPage.login(mattermost.url(), username, password);

        await page.getByRole('button', { name: /Add channels/i }).click();
        await page.getByRole('menuitem', { name: /Create new channel/i }).click();

        const modal = page.locator('#new-channel-modal');
        await expect(modal).toBeVisible({ timeout: 10000 });
        await modal.getByPlaceholder(/Enter a name for your new channel|Channel name/i).fill('boards-from-channel');

        await modal.getByTestId('add-board-to-channel-check').check();
        await modal.locator('[class*="CreateBoardFromTemplate--templates-selector__control"]').click();
        await page.getByText('Empty board', { exact: true }).first().click();

        await modal.getByRole('button', { name: 'Create channel' }).click();
        await expect(modal).not.toBeVisible({ timeout: 15000 });

        const adminClient = await mattermost.getAdminClient();
        const teamId = await getTeamId(mattermost);
        const channel = await adminClient.getChannelByName(teamId, 'boards-from-channel');
        const token = adminClient.getToken();

        await expect.poll(
            async () => (await listTeamBoardsViaApi(mattermost, teamId, token)).some((b) => b.channelId === channel.id),
            { timeout: 20000 },
        ).toBe(true);
    });

    test('channel members get access to a board linked to their channel', async ({ page }) => {
        const { channel } = await createChannel('linked-access', 'Linked Access', [Users.regularUser.username]);
        const board = await seedBoard(mattermost, username, password, 'Channel Access Board', 'P');
        const userClient = await mattermost.getClient(Users.regularUser.username, Users.regularUser.password);
        const beforeLink = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${board.boardId}`, {
            headers: { Authorization: `Bearer ${userClient.getToken()}`, 'X-Requested-With': 'XMLHttpRequest' },
        });
        expect(beforeLink.status).toBe(403);

        await linkBoardToChannelViaApi(mattermost, board.boardId, channel.id, board.token);

        await openBoard(page, mattermost, Users.regularUser.username, Users.regularUser.password, board);

        await expect(page.locator('.ViewTitle .Editable')).toHaveValue('Channel Access Board', { timeout: 15000 });
        await expect(page.locator('.octo-sidebar-list')).toContainText('Channel Access Board', { timeout: 15000 });
    });

    test('the share dialog lists the linked channel and can unlink it', async ({ page }) => {
        const { channel } = await createChannel('linked-share', 'Linked Share', [Users.regularUser.username]);
        const board = await seedBoard(mattermost, username, password, 'Channel Share Board');
        await linkBoardToChannelViaApi(mattermost, board.boardId, channel.id, board.token);

        await openBoard(page, mattermost, username, password, board);
        await page.locator('.ShareBoardButton button').click();
        const dialog = page.locator('.ShareBoardDialog');
        await expect(dialog).toBeVisible({ timeout: 10000 });

        const channelRow = dialog.locator('.channel-item');
        await expect(channelRow).toContainText('Linked Share', { timeout: 10000 });

        await channelRow.locator('.user-item__button').click();
        await page.locator('.Menu.noselect').getByRole('button', { name: 'Unlink' }).click();
        await page.locator('.confirmation-dialog-box').getByRole('button', { name: 'Unlink channel' }).click();

        await expect(dialog.locator('.channel-item')).toHaveCount(0, { timeout: 10000 });
    });
});
