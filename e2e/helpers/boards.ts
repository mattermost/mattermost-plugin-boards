// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import { expect, type Page } from '@playwright/test';

import MattermostContainer from './mmcontainer';
import { MattermostPage } from './mm';

export type SeededBoard = {
    boardId: string;
    teamId: string;
    viewId: string;
    token: string;
};

const apiHeaders = (token: string) => ({
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Requested-With': 'XMLHttpRequest',
});

/** Create a board with a default view through the API and return everything needed to open it. */
export async function seedBoard(
    mattermost: MattermostContainer,
    username: string,
    password: string,
    title: string,
    type: 'O' | 'P' = 'O',
): Promise<SeededBoard> {
    const client = await mattermost.getClient(username, password);
    const token = (client as any).token as string;
    const boardId = await createBoardViaApi(mattermost, title, token, type);
    const { teamId, viewId } = await getBoardMeta(mattermost, boardId, token);
    return { boardId, teamId, viewId, token };
}

export const STATUS_PROPERTY_ID = 'statusprop';

export type SeededKanbanBoard = SeededBoard & { optionIds: Record<string, string> };

/** Replace a board's card property templates through the API. */
export async function setCardPropertiesViaApi(
    mattermost: MattermostContainer,
    boardId: string,
    cardProperties: Array<Record<string, unknown>>,
    token: string,
): Promise<void> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}`, {
        method: 'PATCH',
        headers: apiHeaders(token),
        body: JSON.stringify({ updatedCardProperties: cardProperties }),
    });
    if (!resp.ok) throw new Error(`Failed to set card properties: ${resp.status}`);
}

/**
 * Create a board with a Status select property and a kanban view grouped by it.
 * `optionIds` maps each option name to its id, for seeding card properties.
 */
export async function seedKanbanBoard(
    mattermost: MattermostContainer,
    username: string,
    password: string,
    title: string,
    options: string[] = ['To Do', 'Doing', 'Done'],
    extraProperties: Array<Record<string, unknown>> = [],
    viewFields: Record<string, unknown> = {},
): Promise<SeededKanbanBoard> {
    const client = await mattermost.getClient(username, password);
    const token = (client as any).token as string;
    const boardId = await createBoardViaApi(mattermost, title, token);

    const optionIds: Record<string, string> = {};
    const statusOptions = options.map((value, i) => {
        const id = `opt${i}${value.replace(/[^a-z0-9]/gi, '').toLowerCase()}`;
        optionIds[value] = id;
        return { id, value, color: 'propColorDefault' };
    });
    await setCardPropertiesViaApi(
        mattermost,
        boardId,
        [{ id: STATUS_PROPERTY_ID, name: 'Status', type: 'select', options: statusOptions }, ...extraProperties],
        token,
    );

    const { teamId, viewId } = await getBoardMeta(mattermost, boardId, token, {
        groupById: STATUS_PROPERTY_ID,
        visibleOptionIds: statusOptions.map((o) => o.id),
        ...viewFields,
    });
    return { boardId, teamId, viewId, token, optionIds };
}

/** Create a card through the Focalboard REST API; returns the new card's id. */
export async function createCardViaApi(
    mattermost: MattermostContainer,
    boardId: string,
    title: string,
    token: string,
    properties: Record<string, unknown> = {},
): Promise<string> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/cards`, {
        method: 'POST',
        headers: apiHeaders(token),
        body: JSON.stringify({ title, properties }),
    });
    if (!resp.ok) throw new Error(`Failed to create card: ${resp.status}`);
    return ((await resp.json()) as { id: string }).id;
}

/** Link a board to a channel (or unlink with an empty id) through the API. */
export async function linkBoardToChannelViaApi(
    mattermost: MattermostContainer,
    boardId: string,
    channelId: string,
    token: string,
): Promise<void> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}`, {
        method: 'PATCH',
        headers: apiHeaders(token),
        body: JSON.stringify({ channelId }),
    });
    if (!resp.ok) throw new Error(`Failed to link board to channel: ${resp.status}`);
}

/** List a team's boards visible to the token's user. */
export async function listTeamBoardsViaApi(
    mattermost: MattermostContainer,
    teamId: string,
    token: string,
): Promise<Array<{ id: string; title: string; channelId: string }>> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/teams/${teamId}/boards`, {
        headers: apiHeaders(token),
    });
    if (!resp.ok) throw new Error(`Failed to list boards: ${resp.status}`);
    return (await resp.json()) as Array<{ id: string; title: string; channelId: string }>;
}

/** Soft-delete a board through the API. */
export async function deleteBoardViaApi(
    mattermost: MattermostContainer,
    boardId: string,
    token: string,
): Promise<void> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}`, {
        method: 'DELETE',
        headers: apiHeaders(token),
    });
    if (!resp.ok) throw new Error(`Failed to delete board: ${resp.status}`);
}

/** Restore a soft-deleted board through the API. */
export async function undeleteBoardViaApi(
    mattermost: MattermostContainer,
    boardId: string,
    token: string,
): Promise<void> {
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/undelete`, {
        method: 'POST',
        headers: apiHeaders(token),
    });
    if (!resp.ok) throw new Error(`Failed to undelete board: ${resp.status}`);
}

/** Log in and open a specific board view, waiting for the board to render. */
export async function openBoard(
    page: Page,
    mattermost: MattermostContainer,
    username: string,
    password: string,
    board: Pick<SeededBoard, 'teamId' | 'boardId' | 'viewId'>,
    cardId?: string,
): Promise<void> {
    await page.addInitScript((tid) => {
        localStorage.setItem('lastTeamId', tid);
    }, board.teamId);

    const mmPage = new MattermostPage(page);
    await mmPage.login(mattermost.url(), username, password);
    await page.getByTestId('channel_view').waitFor({ state: 'visible', timeout: 30000 });
    await mmPage.navigateToBoardsFromUrl(mattermost.url());

    const suffix = cardId ? `/${cardId}` : '';
    await page.goto(
        `${mattermost.url()}/boards/team/${board.teamId}/${board.boardId}/${board.viewId}${suffix}`,
        { waitUntil: 'domcontentloaded', timeout: 30000 },
    );
    await expect(page.locator('.BoardComponent')).toBeVisible({ timeout: 15000 });
}

/** Click "New" in the view header, name the card in the dialog, and close it. */
export async function createNamedCardViaUi(page: Page, cardName: string): Promise<void> {
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

/** Open the sidebar options menu for a board. */
export async function openSidebarBoardMenu(page: Page, boardTitle: string): Promise<void> {
    const boardItem = page.locator('.SidebarBoardItem.subitem').filter({ hasText: boardTitle });
    await boardItem.hover();
    await boardItem.locator('.MenuWrapper button.IconButton').click();
    await expect(page.locator('.Menu.noselect')).toBeVisible({ timeout: 5000 });
}

/** Resolve once the webapp's PATCH for a block (any block when `blockId` is omitted) succeeds. */
export function waitForBlockPatch(page: Page, blockId = ''): Promise<unknown> {
    return page.waitForResponse((response) =>
        response.request().method() === 'PATCH' &&
        response.url().includes(`/blocks/${blockId}`) &&
        response.ok());
}

/** Open the card actions menu on a kanban card. */
export async function openKanbanCardMenu(page: Page, cardName: string): Promise<void> {
    const card = page.locator('.KanbanCard').filter({ hasText: cardName });
    await card.hover();
    await card.locator('.CardActionsMenuIcon').click();
    await expect(page.locator('.Menu.noselect')).toBeVisible({ timeout: 5000 });
}

/** Return the first team's id for the default test team. */
export async function getTeamId(mattermost: MattermostContainer): Promise<string> {
    const adminClient = await mattermost.getAdminClient();
    const teams = await adminClient.getMyTeams();
    if (!teams[0]?.id) throw new Error('No team found');
    return teams[0].id;
}

/**
 * Create a board via the Focalboard REST API; returns the new board's id.
 * cardProperties must be [] — Go nil slice serialises as null which crashes
 * getCurrentViewGroupBy when it calls null.find().
 */
export async function createBoardViaApi(
    mattermost: MattermostContainer,
    title: string,
    token: string,
    type: 'O' | 'P' = 'O',
): Promise<string> {
    const teamId = await getTeamId(mattermost);
    const resp = await fetch(`${mattermost.url()}/plugins/focalboard/api/v2/boards`, {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({ teamId, title, type, cardProperties: [] }),
    });
    if (!resp.ok) throw new Error(`Failed to create board: ${resp.status}`);
    return ((await resp.json()) as { id: string }).id;
}

/**
 * Create a default board view block and return the teamId + viewId needed
 * for constructing the full board URL.
 * Boards created via API have no views — this creates one first.
 */
export async function getBoardMeta(
    mattermost: MattermostContainer,
    boardId: string,
    token: string,
    viewFields: Record<string, unknown> = {},
): Promise<{ teamId: string; viewId: string }> {
    const teamId = await getTeamId(mattermost);

    const now = Date.now();
    const viewBlockId = `view${boardId.substring(0, 8)}${now}`.replace(/[^a-z0-9]/gi, '').substring(0, 26);
    const resp = await fetch(
        `${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/blocks`,
        {
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
                    ...viewFields,
                },
            }]),
        },
    );
    if (!resp.ok) throw new Error(`Failed to create board view: ${resp.status}`);
    const created = (await resp.json()) as Array<{ id: string }>;
    return { teamId, viewId: created[0]?.id ?? '' };
}

/**
 * Pre-seed the Focalboard welcomePageViewed config for a user so that the
 * welcome page never appears on first boards visit. Without this, FBRoute
 * redirects to /welcome and the welcome page's goForward() triggers a React
 * rendering error caught by the ErrorBoundary.
 */
export async function seedWelcomePageViewed(
    mattermost: MattermostContainer,
    userId: string,
    token: string,
): Promise<void> {
    const resp = await fetch(
        `${mattermost.url()}/plugins/focalboard/api/v2/users/${userId}/config`,
        {
            method: 'PUT',
            headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            body: JSON.stringify({ updatedFields: { welcomePageViewed: '1' } }),
        },
    );
    if (!resp.ok) throw new Error(`Failed to seed welcomePageViewed: ${resp.status}`);
}

/** Add a user to a board with the given role. */
export async function addBoardMember(
    mattermost: MattermostContainer,
    boardId: string,
    userId: string,
    role: 'admin' | 'editor' | 'commenter' | 'viewer',
    token: string,
): Promise<void> {
    const resp = await fetch(
        `${mattermost.url()}/plugins/focalboard/api/v2/boards/${boardId}/members`,
        {
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
        },
    );
    if (!resp.ok) throw new Error(`Failed to add board member: ${resp.status}`);
}
