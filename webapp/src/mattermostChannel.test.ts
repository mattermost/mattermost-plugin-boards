// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GlobalState} from '@mattermost/types/store'

import {createCurrentChannelSync, getCurrentMattermostChannel} from './mattermostChannel'

const currentUserId = 'admin_id'
const gmChannelId = 'gm_channel_id'

const gmMembers = [
    {id: currentUserId, username: 'admin'},
    {id: 'u1', username: 'gmtester1'},
    {id: 'u2', username: 'gmtester2'},
    {id: 'u3', username: 'gmtester3'},
    {id: 'u4', username: 'gmtester4'},
    {id: 'u5', username: 'gmtester5'},
    {id: 'u6', username: 'gmtester6'},
    {id: 'u7', username: 'gmtester7'},
]

// The server stores a group message display_name truncated to 64 characters,
// which is what the Boards store previously read verbatim.
const truncatedGmDisplayName = 'admin, gmtester1, gmtester2, gmtester3, gmtester4, gmtester5, gm'

// The full member list, with the current user excluded, rebuilt from profiles.
const fullGmDisplayName = 'gmtester1, gmtester2, gmtester3, gmtester4, gmtester5, gmtester6, gmtester7'

type TestChannel = {id: string; name: string; type: string; display_name: string}

const buildState = (channel: TestChannel, {membersLoaded = true} = {}): GlobalState => {
    const profiles: Record<string, {id: string; username: string}> = {}
    const memberIds = new Set<string>()
    for (const member of gmMembers) {
        profiles[member.id] = member
        memberIds.add(member.id)
    }

    return {
        entities: {
            channels: {
                currentChannelId: channel.id,
                channels: {[channel.id]: channel},
            },
            users: {
                currentUserId,
                profiles,
                statuses: {},

                // Mattermost loads channel member profiles asynchronously after a
                // channel becomes active; before that the GM name cannot be rebuilt.
                profilesInChannel: membersLoaded ? {[channel.id]: memberIds} : {},
            },
            general: {
                config: {},
                license: {},
            },
            preferences: {
                myPreferences: {},
            },
        },
    } as unknown as GlobalState
}

describe('mattermostChannel', () => {
    describe('getCurrentMattermostChannel', () => {
        it('expands a truncated group message name to list every member', () => {
            const state = buildState({
                id: gmChannelId,
                name: 'gm_channel',
                type: 'G',
                display_name: truncatedGmDisplayName,
            })

            const channel = getCurrentMattermostChannel(state)

            expect(channel?.id).toBe(gmChannelId)
            expect(channel?.display_name).toBe(fullGmDisplayName)

            // The members dropped by the server-side truncation are now present.
            expect(channel?.display_name).toContain('gmtester6')
            expect(channel?.display_name).toContain('gmtester7')

            // The current user is excluded, matching Mattermost's own GM header.
            expect(channel?.display_name).not.toContain('admin')
        })

        it('keeps the truncated name until member profiles have loaded', () => {
            const state = buildState({
                id: gmChannelId,
                name: 'gm_channel',
                type: 'G',
                display_name: truncatedGmDisplayName,
            }, {membersLoaded: false})

            // Without member profiles the full list cannot be rebuilt yet; the sync
            // re-runs and corrects this once the profiles stream in.
            expect(getCurrentMattermostChannel(state)?.display_name).toBe(truncatedGmDisplayName)
        })

        it('leaves a regular channel display name untouched', () => {
            const state = buildState({
                id: 'public_channel_id',
                name: 'town-square',
                type: 'O',
                display_name: 'Town Square',
            })

            const channel = getCurrentMattermostChannel(state)

            expect(channel?.id).toBe('public_channel_id')
            expect(channel?.display_name).toBe('Town Square')
        })

        it('resolves a direct message name to the other participant', () => {
            const state = buildState({
                id: 'dm_channel_id',
                name: `${currentUserId}__u1`,
                type: 'D',
                display_name: '',
            })

            expect(getCurrentMattermostChannel(state)?.display_name).toBe('gmtester1')
        })

        it('returns undefined when there is no current channel', () => {
            const state = buildState({
                id: gmChannelId,
                name: 'gm_channel',
                type: 'G',
                display_name: truncatedGmDisplayName,
            })
            state.entities.channels.currentChannelId = ''

            expect(getCurrentMattermostChannel(state)).toBeUndefined()
        })
    })

    describe('createCurrentChannelSync', () => {
        const gmChannel: TestChannel = {
            id: gmChannelId,
            name: 'gm_channel',
            type: 'G',
            display_name: truncatedGmDisplayName,
        }

        it('emits the channel on first resolve', () => {
            const state = buildState(gmChannel)
            const onChange = jest.fn()
            const sync = createCurrentChannelSync(() => state, onChange)

            sync()

            expect(onChange).toHaveBeenCalledTimes(1)
            expect(onChange.mock.calls[0][0].id).toBe(gmChannelId)
        })

        it('re-emits when only the display name changes as member profiles load', () => {
            // Start before profiles load: the name is still truncated.
            let state = buildState(gmChannel, {membersLoaded: false})
            const onChange = jest.fn()
            const sync = createCurrentChannelSync(() => state, onChange)

            sync()
            expect(onChange).toHaveBeenCalledTimes(1)
            expect(onChange.mock.calls[0][0].display_name).toBe(truncatedGmDisplayName)

            // Profiles arrive for the same channel id; the full name is resolved.
            state = buildState(gmChannel)
            sync()

            expect(onChange).toHaveBeenCalledTimes(2)
            expect(onChange.mock.calls[1][0].display_name).toBe(fullGmDisplayName)
        })

        it('does not re-emit when neither id nor display name changed', () => {
            let state = buildState(gmChannel)
            const onChange = jest.fn()
            const sync = createCurrentChannelSync(() => state, onChange)

            sync()
            // A new state object with identical resolved channel data.
            state = buildState(gmChannel)
            sync()

            expect(onChange).toHaveBeenCalledTimes(1)
        })

        it('emits again when the active channel switches', () => {
            let state = buildState(gmChannel)
            const onChange = jest.fn()
            const sync = createCurrentChannelSync(() => state, onChange)

            sync()
            state = buildState({
                id: 'public_channel_id',
                name: 'town-square',
                type: 'O',
                display_name: 'Town Square',
            })
            sync()

            expect(onChange).toHaveBeenCalledTimes(2)
            expect(onChange.mock.calls[1][0].id).toBe('public_channel_id')
        })

        it('does not emit when there is no current channel', () => {
            const state = buildState(gmChannel)
            state.entities.channels.currentChannelId = ''
            const onChange = jest.fn()

            createCurrentChannelSync(() => state, onChange)()

            expect(onChange).not.toHaveBeenCalled()
        })
    })
})
