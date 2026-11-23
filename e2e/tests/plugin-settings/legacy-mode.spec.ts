// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import { test, expect, type Page } from '@playwright/test';

import RunContainer from 'helpers/plugincontainer';
import MattermostContainer from 'helpers/mmcontainer';
import { MattermostPage } from 'helpers/mm';
import { Users } from 'helpers/users';
import { openBoard, seedBoard } from 'helpers/boards';

let mattermost: MattermostContainer;

test.beforeAll(async () => {
    test.setTimeout(300000);
    mattermost = await RunContainer();
});

test.afterAll(async () => {
    await mattermost?.stop();
});

// Both suites share one server and toggle the same setting, so they must run in order.
test.describe.configure({ mode: 'serial', timeout: 300000 });

const { username, password } = Users.admin;

async function setLegacyMode(enabled: boolean): Promise<void> {
    const adminClient = await mattermost.getAdminClient();
    const config = await adminClient.getConfig();
    config.PluginSettings.Plugins = {
        ...config.PluginSettings.Plugins,
        focalboard: { ...config.PluginSettings.Plugins?.['focalboard'], legacymode: enabled },
    };
    await adminClient.updateConfig(config);

    await expect.poll(async () => {
        const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/clientConfig`, {
            headers: { Authorization: `Bearer ${adminClient.getToken()}`, 'X-Requested-With': 'XMLHttpRequest' },
        });
        return ((await resp.json()) as { legacyMode?: boolean }).legacyMode ?? false;
    }, { timeout: 15000 }).toBe(enabled);
}

async function boardsBotDisplayName(): Promise<string | undefined> {
    const adminClient = await mattermost.getAdminClient();
    const bots = await adminClient.getBots(0, 200);
    return bots.find((bot) => bot.username === 'boards')?.display_name;
}

async function storedCategoryNames(teamId: string, token: string): Promise<string[]> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/teams/${teamId}/categories`, {
        headers: { Authorization: `Bearer ${token}`, 'X-Requested-With': 'XMLHttpRequest' },
    });
    if (!resp.ok) throw new Error(`Failed to list categories: ${resp.status}`);
    return ((await resp.json()) as Array<{ name: string }>).map((c) => c.name);
}

async function openChannelWithBoardsRHS(page: Page): Promise<void> {
    await MattermostPage.loginAndWait(page, mattermost.url(), username, password);
    await page.getByRole('button', { name: 'focalboard' }).click();
    await expect(page.locator('.RHSChannelBoards')).toBeVisible({ timeout: 15000 });
}

const rhsHeaderTitle = (page: Page) => page.locator('.boards-rhs-header-logo + span');

// The registered product name is rendered as a screen-reader heading in the global header,
// and as a visible label in the product rail on servers that have it.
const productName = (page: Page, name: string) =>
    page.locator('h1.sr-only, .ProductRail__label').filter({ hasText: new RegExp(`^${name}$`) }).first();

const defaultCategoryTitle = (page: Page) =>
    page.locator('.SidebarCategory .category-title').filter({ hasText: /^(Legacy )?Boards$/ });

const labels = {
    original: { name: 'Boards', switcher: 'Find boards', dialogTitle: 'Find Boards' },
    legacy: { name: 'Legacy Boards', switcher: 'Find Legacy Boards', dialogTitle: 'Find Legacy Boards' },
};

/** Checks every surface that carries the product name on a freshly loaded board page. */
async function expectBoardPageLabels(page: Page, expected: typeof labels.original): Promise<void> {
    await expect(productName(page, expected.name)).toBeAttached({ timeout: 15000 });

    await expect(defaultCategoryTitle(page)).toHaveText(expected.name, { timeout: 15000 });
    await expect(defaultCategoryTitle(page)).toHaveAttribute('title', expected.name);

    await expect(page.locator('.BoardsSwitcher')).toHaveText(expected.switcher);
    await page.locator('.BoardsSwitcher').click();
    const dialog = page.locator('.BoardSwitcherDialog');
    await expect(dialog).toBeVisible({ timeout: 10000 });
    await expect(dialog.getByText(expected.dialogTitle, { exact: true })).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(dialog).not.toBeVisible({ timeout: 5000 });
}

test.describe('Legacy Mode: enabling the setting', () => {
    test.beforeAll(async () => {
        await setLegacyMode(false);
    });

    test('the RHS header switches to Legacy Boards without a refresh', async ({ page }) => {
        await openChannelWithBoardsRHS(page);
        await expect(rhsHeaderTitle(page)).toHaveText('Boards', { timeout: 15000 });

        await setLegacyMode(true);

        await expect(rhsHeaderTitle(page)).toHaveText('Legacy Boards', { timeout: 15000 });
        await expect(page.locator('.RHSChannelBoards.empty .empty-paragraph'))
            .toContainText('Legacy Boards is a project management tool');
    });

    test('after a refresh every product label reads Legacy Boards', async ({ page }) => {
        await setLegacyMode(true);
        const board = await seedBoard(mattermost, username, password, 'Legacy Mode On Board');
        await openBoard(page, mattermost, username, password, board);

        await expectBoardPageLabels(page, labels.legacy);
        await expect.poll(boardsBotDisplayName, { timeout: 15000 }).toBe('Legacy Boards');

        // Only the label changes; the stored default category keeps its name.
        expect(await storedCategoryNames(board.teamId, board.token)).toContain('Boards');
        expect(await storedCategoryNames(board.teamId, board.token)).not.toContain('Legacy Boards');
    });
});

test.describe('Legacy Mode: disabling the setting', () => {
    test.beforeAll(async () => {
        await setLegacyMode(true);
    });

    test('the RHS header switches back to Boards without a refresh', async ({ page }) => {
        await openChannelWithBoardsRHS(page);
        await expect(rhsHeaderTitle(page)).toHaveText('Legacy Boards', { timeout: 15000 });

        await setLegacyMode(false);

        await expect(rhsHeaderTitle(page)).toHaveText('Boards', { timeout: 15000 });
        await expect(page.locator('.RHSChannelBoards.empty .empty-paragraph'))
            .toContainText('Boards is a project management tool');
        await expect(page.getByText('Legacy Boards')).toHaveCount(0);
    });

    test('after a refresh every product label is back to Boards', async ({ page }) => {
        await setLegacyMode(false);
        const board = await seedBoard(mattermost, username, password, 'Legacy Mode Off Board');
        await openBoard(page, mattermost, username, password, board);

        await expectBoardPageLabels(page, labels.original);
        await expect(page.getByText('Legacy Boards')).toHaveCount(0);
        await expect.poll(boardsBotDisplayName, { timeout: 15000 }).toBe('Boards');
        expect(await storedCategoryNames(board.teamId, board.token)).toContain('Boards');
    });
});
