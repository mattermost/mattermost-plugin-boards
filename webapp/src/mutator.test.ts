// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import mutator from './mutator'
import octoClient from './octoClient'
import undoManager from './undomanager'
import {TestBlockFactory} from './test/testBlockFactory'
import 'isomorphic-fetch'
import {FetchMock} from './test/fetchMock'
import {mockDOM} from './testUtils'

global.fetch = FetchMock.fn

beforeEach(() => {
    FetchMock.fn.mockReset()
})

beforeAll(() => {
    mockDOM()
})

describe('Mutator', () => {
    test('changePropertyValue', async () => {
        const board = TestBlockFactory.createBoard()
        const card = TestBlockFactory.createCard()
        card.boardId = board.id
        card.fields.properties.property_1 = 'hello'

        await mutator.changePropertyValue(board.id, card, 'property_1', 'hello')

        // No API call should be made as property value DIDN'T CHANGE
        expect(FetchMock.fn).toHaveBeenCalledTimes(0)

        await mutator.changePropertyValue(board.id, card, 'property_1', 'hello world')

        // 1 API call should be made as property value DID CHANGE
        expect(FetchMock.fn).toHaveBeenCalledTimes(1)
    })

    test('changeBoardCardPrefix fails when the server rejects the prefix', async () => {
        const patchBoard = jest.spyOn(octoClient, 'patchBoard').mockResolvedValueOnce({ok: false, status: 400} as Response)
        const undoDescription = undoManager.undoDescription

        await expect(mutator.changeBoardCardPrefix('board-id', '', 'TAKEN')).rejects.toThrow()
        expect(patchBoard).toHaveBeenCalledWith('board-id', {cardPrefix: 'TAKEN'})

        // the failed change must not be added to the undo stack
        expect(undoManager.undoDescription).toBe(undoDescription)

        patchBoard.mockRestore()
    })

    test('duplicateCard', async () => {
        const board = TestBlockFactory.createBoard()
        const card = TestBlockFactory.createCard(board)

        FetchMock.fn.mockReturnValueOnce(FetchMock.jsonResponse(JSON.stringify([card])))
        const [newBlocks, newCardID] = await mutator.duplicateCard(card.id, board.id)

        expect(newBlocks).toHaveLength(1)

        const duplicatedCard = newBlocks[0]
        expect(duplicatedCard.type).toBe('card')
        expect(duplicatedCard.id).toBe(newCardID)
        expect(duplicatedCard.fields.icon).toBe(card.fields.icon)
        expect(duplicatedCard.fields.contentOrder).toHaveLength(card.fields.contentOrder.length)
        expect(duplicatedCard.boardId).toBe(board.id)
    })

    describe('deleteBoard', () => {
        afterEach(() => {
            jest.restoreAllMocks()
        })

        test('runs the afterUndo callback only after the board has been undeleted', async () => {
            const board = TestBlockFactory.createBoard()
            const callOrder: string[] = []

            jest.spyOn(octoClient, 'deleteBoard').mockImplementation(async () => {
                callOrder.push('deleteBoard')
                return {} as Response
            })
            const undeleteBoard = jest.spyOn(octoClient, 'undeleteBoard').mockImplementation(async () => {
                callOrder.push('undeleteBoard')
                return {} as Response
            })

            const afterRedo = jest.fn(async () => {
                callOrder.push('afterRedo')
            })
            const afterUndo = jest.fn(async () => {
                callOrder.push('afterUndo')
            })

            await mutator.deleteBoard(board, 'delete board', afterRedo, afterUndo)
            expect(callOrder).toEqual(['deleteBoard', 'afterRedo'])

            await undoManager.undo()

            // The board must be restored on the server before the afterUndo
            // callback runs. The callback navigates to (and recategorizes) the
            // board, so running it first lands the user on a "Board not found"
            // error page for a board that no longer exists.
            expect(callOrder).toEqual(['deleteBoard', 'afterRedo', 'undeleteBoard', 'afterUndo'])
            expect(undeleteBoard).toHaveBeenCalledWith(board.id)
            expect(afterUndo).toHaveBeenCalledTimes(1)
            expect(afterUndo).toHaveBeenCalledWith(board)
        })
    })
})
