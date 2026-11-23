// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Whether a board was moved to Boards v2.
export type BoardMigration = {
    moved: boolean
    channel_id?: string
    team_name?: string
    channel_name?: string
    channel_display_name?: string
    moved_at?: number
    can_open: boolean
    read_only: boolean
}
