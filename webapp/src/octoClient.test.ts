// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


// Disable console log
console.log = jest.fn()

import {Block} from './blocks/block'
import {createCard} from './blocks/card'
import octoClient from './octoClient'
import 'isomorphic-fetch'
import {FetchMock} from './test/fetchMock'
import {Utils} from './utils'

global.fetch = FetchMock.fn

beforeEach(() => {
    FetchMock.fn.mockReset()
})

test('OctoClient: get blocks', async () => {
    const blocks = createBlocks()

    FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify(blocks)))
    let boards = await octoClient.getBlocksWithType('card')
    expect(boards.length).toBe(blocks.length)

    FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify(blocks)))
    let response = await octoClient.exportBoardArchive('board')
    expect(response.status).toBe(200)

    FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify(blocks)))
    response = await octoClient.exportFullArchive('team')
    expect(response.status).toBe(200)

    FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify(blocks)))
    const parentId = 'id1'
    boards = await octoClient.getBlocksWithParent(parentId)
    expect(boards.length).toBe(blocks.length)

    FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify(blocks)))
    boards = await octoClient.getBlocksWithParent(parentId, 'card')
    expect(boards.length).toBe(blocks.length)
})

test('OctoClient: insert blocks', async () => {
    const blocks = createBlocks()

    await octoClient.insertBlocks('board-id', blocks)

    expect(FetchMock.fn).toHaveBeenCalledTimes(1)
    expect(FetchMock.fn).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({
            method: 'POST',
            body: JSON.stringify(blocks),
        }))
})

test('OctoClient: importFullArchive', async () => {
    const archive = new File([''], 'test')

    await octoClient.importFullArchive(archive)

    expect(FetchMock.fn).toHaveBeenCalledTimes(1)
    expect(FetchMock.fn).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({
            method: 'POST',
        }))
})

const statusResponse = (status: number): Promise<Response> =>
    Promise.resolve(new Response('', {status}))

describe('OctoClient: getBoard vs boardNotFound', () => {
    const boardID = 'board-id-1'
    const boardPath = `http://localhost/api/v2/boards/${boardID}`

    test('getBoard returns the board on a 200 response', async () => {
        const board = {id: boardID, title: 'My Board'}
        FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify(board)))

        const result = await octoClient.getBoard(boardID)

        expect(result).toEqual(board)
        expect(FetchMock.fn).toHaveBeenCalledWith(
            boardPath,
            expect.objectContaining({method: 'GET'}),
        )
    })

    // getBoard collapses every non-200 into undefined, so on its own it cannot
    // tell a deleted board (404) apart from a private board the caller may not
    // read (403). This is the ambiguity that hid the access-denied page.
    test.each([403, 404, 401, 500])('getBoard returns undefined on a %i response', async (status) => {
        FetchMock.fn.mockReturnValueOnce(statusResponse(status))

        await expect(octoClient.getBoard(boardID)).resolves.toBeUndefined()
    })

    test('boardNotFound is true only for a 404 (board was deleted)', async () => {
        FetchMock.fn.mockReturnValueOnce(statusResponse(404))

        await expect(octoClient.boardNotFound(boardID)).resolves.toBe(true)
    })

    test('boardNotFound is false for a 403 (private board the user cannot read)', async () => {
        FetchMock.fn.mockReturnValueOnce(statusResponse(403))

        await expect(octoClient.boardNotFound(boardID)).resolves.toBe(false)
    })

    test.each([200, 401, 500])('boardNotFound is false for a %i response', async (status) => {
        FetchMock.fn.mockReturnValueOnce(statusResponse(status))

        await expect(octoClient.boardNotFound(boardID)).resolves.toBe(false)
    })

    test('boardNotFound forwards the share read_token on the probe request', async () => {
        const readTokenSpy = jest.spyOn(Utils, 'getReadToken').mockReturnValue('secret-token')
        FetchMock.fn.mockReturnValueOnce(statusResponse(404))

        await octoClient.boardNotFound(boardID)

        expect(FetchMock.fn).toHaveBeenCalledWith(
            `${boardPath}?read_token=secret-token`,
            expect.objectContaining({method: 'GET'}),
        )
        readTokenSpy.mockRestore()
    })

    test('getBoard forwards the share read_token too (shared with boardNotFound)', async () => {
        const readTokenSpy = jest.spyOn(Utils, 'getReadToken').mockReturnValue('secret-token')
        FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify({id: boardID})))

        await octoClient.getBoard(boardID)

        expect(FetchMock.fn).toHaveBeenCalledWith(
            `${boardPath}?read_token=secret-token`,
            expect.objectContaining({method: 'GET'}),
        )
        readTokenSpy.mockRestore()
    })
})

function createBlocks(): Block[] {
    const blocks = []

    for (let i = 0; i < 5; i++) {
        const block = createCard()
        block.id = `block${i + 1}`
        blocks.push(block)
    }

    return blocks
}

test('OctoClient: GetFileInfo', async () => {
    FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify({
        name: 'test.txt',
        size: 2300,
        extension: '.txt',
    })))
    await octoClient.getFileInfo('board-id', 'file-id')
    expect(FetchMock.fn).toHaveBeenCalledTimes(1)
    expect(FetchMock.fn).toHaveBeenCalledWith(
        'http://localhost/api/v2/files/teams/0/board-id/file-id/info',
        expect.objectContaining({
            headers: {
                Accept: 'application/json',
                Authorization: '',
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            }}))
})
