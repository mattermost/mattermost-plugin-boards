// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import MattermostContainer from './mmcontainer';

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
