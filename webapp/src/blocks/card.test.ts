// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {TestBlockFactory} from '../test/testBlockFactory'

import {createCard, formatTicketCode} from './card'

describe('card tests', () => {
    it('formatTicketCode should combine prefix and number', () => {
        expect(formatTicketCode('PROJ', 42)).toBe('PROJ-42')
    })

    it('formatTicketCode should return empty without a prefix or number', () => {
        expect(formatTicketCode('', 42)).toBe('')
        expect(formatTicketCode(undefined, 42)).toBe('')
        expect(formatTicketCode('PROJ', 0)).toBe('')
        expect(formatTicketCode('PROJ', undefined)).toBe('')
    })

    it('createCard should preserve ticketCode and cardNumber', () => {
        const card = TestBlockFactory.createCard()
        card.fields.cardNumber = 7
        card.ticketCode = 'PROJ-7'

        const copy = createCard(card)
        expect(copy.ticketCode).toBe('PROJ-7')
        expect(copy.fields.cardNumber).toBe(7)
        expect(copy.cardNumber).toBe(7)
    })

    it('createCard should not set a card number on a new card', () => {
        const card = createCard()
        expect('cardNumber' in card.fields).toBe(false)
    })
})
