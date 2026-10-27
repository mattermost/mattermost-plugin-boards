// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import {Block, createBlock} from './block'

type CardFields = {
    icon?: string
    isTemplate?: boolean
    properties: Record<string, string | string[]>
    contentOrder: Array<string | string[]>
    cardNumber?: number
}

type Card = Block & {
    fields: CardFields
    cardNumber?: number
    ticketCode?: string
}

function createCard(block?: Block | Card): Card {
    const contentOrder: Array<string|string[]> = []
    const contentIds = block?.fields?.contentOrder?.filter((id: any) => id !== null)

    if (contentIds?.length > 0) {
        for (const contentId of contentIds) {
            if (typeof contentId === 'string') {
                contentOrder.push(contentId)
            } else {
                contentOrder.push(contentId.slice())
            }
        }
    }
    const card: Card = {
        ...createBlock(block),
        type: 'card',
        fields: {
            icon: block?.fields.icon || '',
            properties: {...(block?.fields.properties || {})},
            contentOrder,
            isTemplate: block?.fields.isTemplate || false,
        },
        cardNumber: block?.fields.cardNumber || 0,
    }

    // cardNumber is assigned by the server; only carry over an existing one
    if (block?.fields.cardNumber) {
        card.fields.cardNumber = block.fields.cardNumber
    }
    if (block && 'ticketCode' in block && block.ticketCode) {
        card.ticketCode = block.ticketCode
    }
    return card
}

// formatTicketCode builds a card's ticket code (e.g. PROJ-42), or returns ''
// when the board has no prefix or the card has no number yet.
function formatTicketCode(cardPrefix?: string, cardNumber?: number): string {
    if (!cardPrefix || !cardNumber) {
        return ''
    }
    return `${cardPrefix}-${cardNumber}`
}

export {Card, createCard, formatTicketCode}
