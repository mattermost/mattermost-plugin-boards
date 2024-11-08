import {Channel} from 'mattermost-redux/types/channels'
import configureStore from 'redux-mock-store'

function getChannelMock(override?: Partial<Channel>): Channel {
    const defaultChannel: Channel = {
        id: 'channel_id',
        create_at: 0,
        update_at: 0,
        delete_at: 0,
        team_id: 'team_id',
        type: 'O',
        display_name: 'name',
        name: 'DN',
        header: 'header',
        purpose: 'purpose',
        last_post_at: 0,
        creator_id: 'id',
        scheme_id: 'id',
        group_constrained: false,
        total_msg_count: 0,
        extra_update_at: 0
    }
    return Object.assign({}, defaultChannel, override)
}

const mockStore = configureStore([])
export const mockMMStore = mockStore({
    entities: {
        channels: {
            currentChannelId: 'current_channel_id',
            myMembers: {
                current_channel_id: {
                    channel_id: 'current_channel_id',
                    user_id: 'current_user_id',
                    roles: 'channel_role',
                    mention_count: 1,
                    msg_count: 9,
                },
            },
            channels: {
                current_channel_id: getChannelMock({
                    id: 'current_channel_id',
                    name: 'default-name',
                    display_name: 'Default',
                    delete_at: 0,
                    type: 'O',
                    team_id: 'team_id',
                }),
                current_user_id__existingId: getChannelMock({
                    id: 'current_user_id__existingId',
                    name: 'current_user_id__existingId',
                    display_name: 'Default',
                    delete_at: 0,
                    type: 'O',
                    team_id: 'team_id',
                }),
            },
            channelsInTeam: {
                'team-id': new Set(['asdf']),
            },
            messageCounts: {
                current_channel_id: {total: 10},
                current_user_id__existingId: {total: 0},
            },
        },
        teams: {
            currentTeamId: 'team-id',
            teams: {
                'team-id': {
                    id: 'team_id',
                    name: 'team-1',
                    display_name: 'Team 1',
                },
            },
            myMembers: {
                'team-id': {roles: 'team_role'},
            },
        },
        users: {
            currentUserId: 'current_user_id',
            profiles: {
                current_user_id: {roles: 'system_role'},
            },
        },
        preferences: {
            myPreferences: {
                'display_settings--name_format': {
                    category: 'display_settings',
                    name: 'name_format',
                    user_id: 'current_user_id',
                    value: 'username',
                },
            },
        },
        roles: {
            roles: {
                system_role: {
                    permissions: [],
                },
                team_role: {
                    permissions: [],
                },
                channel_role: {
                    permissions: [],
                },
            },
        },
        general: {
            license: {IsLicensed: 'false'},
            serverVersion: '5.4.0',
            config: {PostEditTimeLimit: '-1'},
        },
    },
})
