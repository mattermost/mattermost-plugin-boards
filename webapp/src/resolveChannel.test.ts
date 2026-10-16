// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GlobalState} from '@mattermost/types/store'

import {resolveChannelForBoards} from './resolveChannel'

const currentUserId = 'user-current'
const memberIds = [
    'user-1',
    'user-2',
    'user-3',
    'user-4',
    'user-5',
    'user-6',
    'user-7',
]

const fullMemberUsernames = [
    'admin',
    'boarduser_member_1',
    'boarduser_member_2',
    'boarduser_member_3',
    'boarduser_member_4',
    'boarduser_member_5',
    'boarduser_member_6',
    'boarduser_member_7',
]

const truncatedDisplayName = fullMemberUsernames.join(', ').slice(0, 64)

const profiles: Record<string, {id: string; username: string}> = {
    [currentUserId]: {id: currentUserId, username: 'admin'},
}
memberIds.forEach((id, index) => {
    profiles[id] = {id, username: `boarduser_member_${index + 1}`}
})

function mmState(overrides: {
    channelId?: string
    type?: string
    displayName?: string
    profilesInChannel?: Set<string>
}): GlobalState {
    const channelId = overrides.channelId || 'gm-channel-id'
    const channel = {
        id: channelId,
        name: 'group-message',
        display_name: overrides.displayName ?? truncatedDisplayName,
        type: overrides.type || 'G',
    }

    return {
        entities: {
            users: {
                currentUserId,
                profiles,
                profilesInChannel: overrides.profilesInChannel === undefined ? {} : {
                    [channelId]: overrides.profilesInChannel,
                },
            },
            channels: {
                channels: {
                    [channelId]: channel,
                },
            },
            preferences: {
                myPreferences: {},
            },
            general: {
                config: {},
                license: {},
            },
        },
    } as unknown as GlobalState
}

describe('resolveChannelForBoards', () => {
    it('rebuilds a group-message display name from members when the stored name is truncated', () => {
        const channelId = 'gm-channel-id'
        const resolved = resolveChannelForBoards(
            mmState({
                channelId,
                profilesInChannel: new Set([currentUserId, ...memberIds]),
            }),
            channelId,
        )

        expect(resolved).toBeDefined()
        expect(resolved!.display_name).not.toEqual(truncatedDisplayName)
        expect(resolved!.display_name).toContain('boarduser_member_7')
        expect(resolved!.display_name).not.toContain('admin')
        memberIds.forEach((_, index) => {
            expect(resolved!.display_name).toContain(`boarduser_member_${index + 1}`)
        })
    })

    it('falls back to the stored truncated name when member profiles are not loaded', () => {
        const channelId = 'gm-unresolved'
        const resolved = resolveChannelForBoards(mmState({channelId}), channelId)

        expect(resolved).toBeDefined()
        expect(resolved!.display_name).toEqual(truncatedDisplayName)
        expect(resolved!.display_name).not.toContain('boarduser_member_7')
    })

    it('leaves a public channel display name unchanged', () => {
        const channelId = 'town-square'
        const resolved = resolveChannelForBoards(
            mmState({
                channelId,
                type: 'O',
                displayName: 'Town Square',
            }),
            channelId,
        )

        expect(resolved!.display_name).toEqual('Town Square')
    })

    it('returns undefined when the channel id is missing', () => {
        expect(resolveChannelForBoards(mmState({}), '')).toBeUndefined()
        expect(resolveChannelForBoards(mmState({}), undefined)).toBeUndefined()
    })
})
