// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {RelationType, getRelationTypeDisplayName, createCardRelation} from '../../blocks/cardRelation'
import {Card} from '../../blocks/card'
import {Block} from '../../blocks/block'
import octoClient from '../../octoClient'
import Dialog from '../dialog'
import Button from '../../widgets/buttons/button'

import './addRelationDialog.scss'

type Props = {
    card: Card
    boardId: string
    onClose: () => void
    onRelationCreated: () => void
}

const RELATION_TYPES: RelationType[] = [
    RelationType.RelatesTo,
    RelationType.Blocks,
    RelationType.BlockedBy,
    RelationType.Duplicates,
    RelationType.Clones,
    RelationType.Causes,
]

const AddRelationDialog = (props: Props): JSX.Element => {
    const {card, boardId, onClose, onRelationCreated} = props
    const intl = useIntl()

    const [searchQuery, setSearchQuery] = useState('')
    const [searchResults, setSearchResults] = useState<Card[]>([])
    const [selectedCard, setSelectedCard] = useState<Card | null>(null)
    const [relationType, setRelationType] = useState<RelationType>(RelationType.RelatesTo)
    const [loading, setLoading] = useState(false)
    const [creating, setCreating] = useState(false)
    const [error, setError] = useState<string | null>(null)

    // Load all cards from the board for selection
    // Re-load when card.id changes to ensure current card is always excluded
    useEffect(() => {
        loadBoardCards()
    }, [boardId, card.id])

    const loadBoardCards = async () => {
        try {
            setLoading(true)
            const blocks = await octoClient.getAllBlocks(boardId)
            // Filter to only cards and exclude current card
            const cards = (blocks as Block[]).filter((b): b is Card => b.type === 'card' && b.id !== card.id) as Card[]
            setSearchResults(cards)
        } catch (err) {
            console.error('Failed to load cards:', err)
            setError(intl.formatMessage({
                id: 'AddRelationDialog.error.loadFailed',
                defaultMessage: 'Failed to load cards',
            }))
        } finally {
            setLoading(false)
        }
    }

    const filteredCards = searchResults.filter((c) => {
        const query = searchQuery.toLowerCase()
        const titleMatch = c.title.toLowerCase().includes(query)
        const codeMatch = c.code?.toLowerCase().includes(query)
        return titleMatch || codeMatch
    })

    const handleCreate = useCallback(async () => {
        if (!selectedCard) {
            return
        }

        try {
            setCreating(true)
            setError(null)
            const relation = createCardRelation({
                sourceCardId: card.id,
                targetCardId: selectedCard.id,
                relationType,
                boardId,
            })
            await octoClient.createCardRelation(card.id, relation)
            onRelationCreated()
            onClose()
        } catch (err) {
            console.error('Failed to create relation:', err)
            setError(intl.formatMessage({
                id: 'AddRelationDialog.error.createFailed',
                defaultMessage: 'Failed to create relation',
            }))
        } finally {
            setCreating(false)
        }
    }, [card.id, selectedCard, relationType, boardId, onRelationCreated, onClose])

    return (
        <Dialog
            onClose={onClose}
            toolsMenu={null}
        >
            <div className='AddRelationDialog'>
                <div className='AddRelationDialog__header'>
                    <h3>
                        <FormattedMessage
                            id='AddRelationDialog.title'
                            defaultMessage='Add Relation'
                        />
                    </h3>
                </div>

                <div className='AddRelationDialog__content'>
                    {/* Relation Type Selector */}
                    <div className='AddRelationDialog__section'>
                        <label className='AddRelationDialog__label'>
                            <FormattedMessage
                                id='AddRelationDialog.relationType'
                                defaultMessage='Relation Type'
                            />
                        </label>
                        <select
                            className='AddRelationDialog__select'
                            value={relationType}
                            onChange={(e) => setRelationType(e.target.value as RelationType)}
                        >
                            {RELATION_TYPES.map((type) => (
                                <option
                                    key={type}
                                    value={type}
                                >
                                    {getRelationTypeDisplayName(type)}
                                </option>
                            ))}
                        </select>
                    </div>

                    {/* Card Search */}
                    <div className='AddRelationDialog__section'>
                        <label className='AddRelationDialog__label'>
                            <FormattedMessage
                                id='AddRelationDialog.selectCard'
                                defaultMessage='Select Card'
                            />
                        </label>
                        <input
                            type='text'
                            className='AddRelationDialog__search'
                            placeholder={intl.formatMessage({
                                id: 'AddRelationDialog.searchPlaceholder',
                                defaultMessage: 'Search cards...',
                            })}
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                        />
                    </div>

                    {/* Card List */}
                    <div className='AddRelationDialog__cards'>
                        {loading && (
                            <div className='AddRelationDialog__loading'>
                                <FormattedMessage
                                    id='AddRelationDialog.loading'
                                    defaultMessage='Loading cards...'
                                />
                            </div>
                        )}
                        {!loading && filteredCards.length === 0 && (
                            <div className='AddRelationDialog__empty'>
                                <FormattedMessage
                                    id='AddRelationDialog.noCards'
                                    defaultMessage='No cards found'
                                />
                            </div>
                        )}
                        {!loading && filteredCards.map((c) => (
                            <div
                                key={c.id}
                                className={`AddRelationDialog__card ${selectedCard?.id === c.id ? 'AddRelationDialog__card--selected' : ''}`}
                                onClick={() => setSelectedCard(c)}
                                role='option'
                                aria-selected={selectedCard?.id === c.id}
                                tabIndex={0}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault()
                                        setSelectedCard(c)
                                    }
                                }}
                            >
                                {c.code && (
                                    <span className='AddRelationDialog__card-code'>
                                        {c.code}
                                    </span>
                                )}
                                <span className='AddRelationDialog__card-title'>
                                    {c.title || 'Untitled'}
                                </span>
                            </div>
                        ))}
                    </div>

                    {error && (
                        <div className='AddRelationDialog__error'>
                            {error}
                        </div>
                    )}
                </div>

                <div className='AddRelationDialog__footer'>
                    <Button
                        onClick={onClose}
                        emphasis='tertiary'
                    >
                        <FormattedMessage
                            id='AddRelationDialog.cancel'
                            defaultMessage='Cancel'
                        />
                    </Button>
                    <Button
                        onClick={handleCreate}
                        emphasis='primary'
                        disabled={!selectedCard || creating}
                    >
                        {creating ? (
                            <FormattedMessage
                                id='AddRelationDialog.creating'
                                defaultMessage='Creating...'
                            />
                        ) : (
                            <FormattedMessage
                                id='AddRelationDialog.create'
                                defaultMessage='Create Relation'
                            />
                        )}
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default AddRelationDialog
