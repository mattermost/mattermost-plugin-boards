// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {act, render} from '@testing-library/react'

import configureStore from 'redux-mock-store'

import {Provider as ReduxProvider} from 'react-redux'

import {mocked} from 'jest-mock'

import {createBoardView} from '../../blocks/boardView'

import {Utils} from '../../utils'
import {createCard} from '../../blocks/card'
import {createBoard} from '../../blocks/board'
import octoClient from '../../octoClient'
import {wrapIntl} from '../../testUtils'

import BoardsUnfurl from './boardsUnfurl'

jest.mock('../../octoClient')
jest.mock('../../utils')
const mockedOctoClient = mocked(octoClient, true)
const mockedUtils = mocked(Utils, true)
mockedUtils.createGuid = jest.requireActual('../../utils').Utils.createGuid
mockedUtils.blockTypeToIDType = jest.requireActual('../../utils').Utils.blockTypeToIDType
mockedUtils.displayDateTime = jest.requireActual('../../utils').Utils.displayDateTime

let mockDisplayDateTime: jest.SpyInstance

beforeEach(() => {
    mockDisplayDateTime = jest.spyOn(Utils, 'displayDateTime').mockImplementation(() => 'July 10, 2022 at 1:40 AM')
})

afterEach(() => {
    mockDisplayDateTime.mockRestore()
})

describe('components/boardsUnfurl/BoardsUnfurl', () => {
    const team = {
        id: 'team-id',
        name: 'team',
        display_name: 'Team name',
    }

    beforeEach(() => {
        // This is done to the websocket not to try to connect directly
        mockedUtils.isFocalboardPlugin.mockReturnValue(true)
        jest.clearAllMocks()
    })

    it('renders normally', async () => {
        const mockStore = configureStore([])
        const store = mockStore({
            language: {
                value: 'en',
            },
            teams: {
                allTeams: [team],
                current: team,
            },
        })

        const cards = [{...createCard(), title: 'test card', updateAt: 12345}]
        const board = {...createBoard(), title: 'test board'}

        mockedOctoClient.getBlocksWithBlockID.mockResolvedValueOnce(cards)
        mockedOctoClient.getBoard.mockResolvedValueOnce(board)

        const component = (
            <ReduxProvider store={store}>
                {wrapIntl(
                    <BoardsUnfurl
                        embed={{data: JSON.stringify({workspaceID: "foo", cardID: cards[0].id, boardID: board.id, readToken: "abc", originalPath: "/test"})}}
                    />,
                )}
            </ReduxProvider>
        )

        let container: Element | DocumentFragment | null = null

        await act(async () => {
            const result = render(component)
            container = result.container
        })
        expect(mockedOctoClient.getBoard).toBeCalledWith(board.id)
        expect(mockedOctoClient.getBlocksWithBlockID).toBeCalledWith(cards[0].id, board.id, "abc")

        expect(container).toMatchSnapshot()
    })

    it('renders when limited', async () => {
        const mockStore = configureStore([])
        const store = mockStore({
            language: {
                value: 'en',
            },
            teams: {
                allTeams: [team],
                current: team,
            },
        })

        const cards = [{...createCard(), title: 'test card', limited: true, updateAt: 12345}]
        const board = {...createBoard(), title: 'test board'}

        mockedOctoClient.getBlocksWithBlockID.mockResolvedValueOnce(cards)
        mockedOctoClient.getBoard.mockResolvedValueOnce(board)

        const component = (
            <ReduxProvider store={store}>
                {wrapIntl(
                    <BoardsUnfurl
                        embed={{data: JSON.stringify({workspaceID: "foo", cardID: cards[0].id, boardID: board.id, readToken: "abc", originalPath: "/test"})}}
                    />,
                )}
            </ReduxProvider>
        )

        let container: Element | DocumentFragment | null = null

        await act(async () => {
            const result = render(component)
            container = result.container
        })

        expect(container).toMatchSnapshot()
    })

    it('test no card', async () => {
        const mockStore = configureStore([])
        const store = mockStore({
            language: {
                value: 'en',
            },
            teams: {
                allTeams: [team],
                current: team,
            },
        })

        const board = {...createBoard(), title: 'test board'}
        // mockedOctoClient.getBoard.mockResolvedValueOnce(board)

        const component = (
            <ReduxProvider store={store}>
                {wrapIntl(
                    <BoardsUnfurl
                        embed={{data: JSON.stringify({workspaceID: 'foo', cardID: '', boardID: board.id, readToken: 'abc', originalPath: '/test'})}}
                    />,
                )}
            </ReduxProvider>
        )

        let container: Element | DocumentFragment | null = null

        await act(async () => {
            const result = render(component)
            container = result.container
        })
        expect(container).toMatchSnapshot()
    })

    it('test invalid card, valid block', async () => {
        const mockStore = configureStore([])
        const store = mockStore({
            language: {
                value: 'en',
            },
            teams: {
                allTeams: [team],
                current: team,
            },
        })

        const cards = [{...createBoardView(), title: 'test view', updateAt: 12345}]
        const board = {...createBoard(), title: 'test board'}

        mockedOctoClient.getBlocksWithBlockID.mockResolvedValueOnce(cards)
        mockedOctoClient.getBoard.mockResolvedValueOnce(board)

        const component = (
            <ReduxProvider store={store}>
                {wrapIntl(
                    <BoardsUnfurl
                        embed={{data: JSON.stringify({workspaceID: 'foo', cardID: cards[0].id, boardID: board.id, readToken: 'abc', originalPath: '/test'})}}
                    />,
                )}
            </ReduxProvider>
        )

        let container: Element | DocumentFragment | null = null

        await act(async () => {
            const result = render(component)
            container = result.container
        })
        expect(mockedOctoClient.getBoard).toBeCalledWith(board.id)
        expect(mockedOctoClient.getBlocksWithBlockID).toBeCalledWith(cards[0].id, board.id, 'abc')

        expect(container).toMatchSnapshot()
    })

    it('test invalid card, invalid block', async () => {
        const mockStore = configureStore([])
        const store = mockStore({
            language: {
                value: 'en',
            },
            teams: {
                allTeams: [team],
                current: team,
            },
        })

        const board = {...createBoard(), title: 'test board'}

        mockedOctoClient.getBlocksWithBlockID.mockResolvedValueOnce([])
        mockedOctoClient.getBoard.mockResolvedValueOnce(board)

        const component = (
            <ReduxProvider store={store}>
                {wrapIntl(
                    <BoardsUnfurl
                        embed={{data: JSON.stringify({workspaceID: 'foo', cardID: 'invalidCard', boardID: board.id, readToken: 'abc', originalPath: '/test'})}}
                    />,
                )}
            </ReduxProvider>
        )

        let container: Element | DocumentFragment | null = null

        await act(async () => {
            const result = render(component)
            container = result.container
        })
        expect(mockedOctoClient.getBoard).toBeCalledWith(board.id)
        expect(mockedOctoClient.getBlocksWithBlockID).toBeCalledWith('invalidCard', board.id, 'abc')

        expect(container).toMatchSnapshot()
    })
})

