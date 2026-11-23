// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {Provider as ReduxProvider} from 'react-redux'
import {renderHook} from '@testing-library/react'

import {Permission} from '../constants'
import {mockStateStore} from '../testUtils'

import {useHasPermissions} from './permissions'

const board = {id: 'board-1', minimumRole: ''}

const renderPermissions = (
    permissions: Permission[],
    migration?: {moved: boolean, read_only: boolean},
    {legacyMode = true, movedBoardsReadOnly = true} = {},
) => {
    const store = mockStateStore([], {
        clientConfig: {value: {legacyMode, movedBoardsReadOnly}},
        boards: {
            boards: {[board.id]: board},
            templates: {},
            myBoardMemberships: {[board.id]: {userId: 'user-1', schemeAdmin: true}},
        },
        boardMigration: migration ? {byBoardId: {[board.id]: {...migration, can_open: true}}} : undefined,
    })
    const wrapper = ({children}: {children: React.ReactNode}) => <ReduxProvider store={store}>{children}</ReduxProvider>
    return renderHook(() => useHasPermissions('team-1', board.id, permissions), {wrapper}).result.current
}

describe('hooks/permissions', () => {
    const writePermissions = [Permission.ManageBoardCards, Permission.ManageBoardProperties, Permission.CommentBoardCards, Permission.ShareBoard, Permission.DeleteBoard]

    it('grants every permission to an admin of a board that did not move', () => {
        for (const permission of [Permission.ViewBoard, ...writePermissions]) {
            expect(renderPermissions([permission], {moved: false, read_only: false})).toBe(true)
        }
    })

    it('grants permissions when the board has no migration info', () => {
        expect(renderPermissions([Permission.ManageBoardCards])).toBe(true)
    })

    it('grants write permissions on a moved board that is not read-only', () => {
        expect(renderPermissions([Permission.ManageBoardCards], {moved: true, read_only: false})).toBe(true)
    })

    it('only grants viewing on a moved board that is read-only', () => {
        expect(renderPermissions([Permission.ViewBoard], {moved: true, read_only: true})).toBe(true)
        for (const permission of writePermissions) {
            expect(renderPermissions([permission], {moved: true, read_only: true})).toBe(false)
        }
    })

    it('ignores a read-only moved board outside Legacy Mode', () => {
        for (const permission of writePermissions) {
            expect(renderPermissions([permission], {moved: true, read_only: true}, {legacyMode: false})).toBe(true)
        }
    })

    it('ignores a read-only moved board when the setting is off', () => {
        for (const permission of writePermissions) {
            expect(renderPermissions([permission], {moved: true, read_only: true}, {movedBoardsReadOnly: false})).toBe(true)
        }
    })
})
