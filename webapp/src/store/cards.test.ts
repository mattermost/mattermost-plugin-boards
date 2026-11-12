// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {TestBlockFactory} from '../test/testBlockFactory'

import {getCurrentViewCardsSortedFilteredAndGroupedWithoutLimit} from './cards'

import {RootState} from './index'

describe('store/cards', () => {
    test('sorting a text property survives values left over from a property type change', () => {
        const board = TestBlockFactory.createBoard()
        board.cardProperties = [{id: 'text', name: 'Text', type: 'text', options: []}]

        const view = TestBlockFactory.createTableView(board)
        view.fields.filter.filters = []
        view.fields.sortOptions = [{propertyId: 'text', reversed: false}]

        const values = [['b', 'c'], 'a', ['d']]
        const cards = values.map((value, i) => {
            const card = TestBlockFactory.createCard(board)
            card.id = `card${i}`
            card.fields.properties = {text: value}
            return card
        })

        const state = {
            boards: {current: board.id, boards: {[board.id]: board}, templates: {}},
            views: {current: view.id, views: {[view.id]: view}},
            cards: {cards: Object.fromEntries(cards.map((c) => [c.id, c])), templates: {}},
            comments: {commentsByCard: {}},
            searchText: {value: ''},
            users: {boardUsers: {}},
        } as unknown as RootState

        const sorted = getCurrentViewCardsSortedFilteredAndGroupedWithoutLimit(state)

        expect(sorted.map((c) => c.id)).toEqual(['card1', 'card0', 'card2'])
    })
})
