// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'
import {useHistory, useRouteMatch, generatePath} from 'react-router-dom'

import {CardRelation, getRelationTypeDisplayName, getInverseRelationType} from '../../blocks/cardRelation'
import {Card} from '../../blocks/card'
import octoClient from '../../octoClient'
import {Utils} from '../../utils'
import {sendFlashMessage} from '../flashMessages'
import ConfirmationDialogBox, {ConfirmationDialogBoxProps} from '../confirmationDialogBox'

import AddRelationDialog from './addRelationDialog'

import './cardRelations.scss'

type Props = {
    card: Card
    boardId: string
    readonly: boolean
}

type RelatedCardInfo = {
    id: string
    title: string
    icon: string
}

const CardRelations = (props: Props): JSX.Element => {
    const {card, boardId, readonly} = props
    const intl = useIntl()
    const history = useHistory()
    const match = useRouteMatch<{boardId: string, viewId?: string, cardId?: string, teamId?: string}>()
    const [relations, setRelations] = useState<CardRelation[]>([])
    const [relatedCards, setRelatedCards] = useState<Map<string, RelatedCardInfo>>(new Map())
    const [loading, setLoading] = useState(true)
    const [showAddDialog, setShowAddDialog] = useState(false)
    const [confirmDelete, setConfirmDelete] = useState<ConfirmationDialogBoxProps | null>(null)

    const loadRelations = useCallback(async () => {
        try {
            setLoading(true)
            const cardRelations = await octoClient.getCardRelations(card.id)
            setRelations(cardRelations || [])

            // Load related card info
            if (cardRelations && cardRelations.length > 0) {
                const cardIds = new Set<string>()
                cardRelations.forEach((r) => {
                    if (r.sourceCardId !== card.id) {
                        cardIds.add(r.sourceCardId)
                    }
                    if (r.targetCardId !== card.id) {
                        cardIds.add(r.targetCardId)
                    }
                })

                // Fetch all cards from the board and filter
                const allBlocks = await octoClient.getAllBlocks(boardId)
                const allCards = allBlocks.filter((b): b is Card => b.type === 'card') as Card[]
                const cardMap = new Map<string, RelatedCardInfo>()
                allCards.forEach((c) => {
                    if (cardIds.has(c.id)) {
                        cardMap.set(c.id, {
                            id: c.id,
                            title: c.title || 'Untitled',
                            icon: c.fields.icon || '📄',
                        })
                    }
                })
                setRelatedCards(cardMap)
            }
        } catch (error) {
            console.error('Failed to load card relations:', error)
        } finally {
            setLoading(false)
        }
    }, [card.id, boardId])

    useEffect(() => {
        loadRelations()
    }, [loadRelations])

    const handleDeleteRelation = useCallback(async (relation: CardRelation) => {
        const relatedCardId = relation.sourceCardId === card.id ? relation.targetCardId : relation.sourceCardId
        const relatedCard = relatedCards.get(relatedCardId)
        const cardTitle = relatedCard?.title || relatedCardId

        setConfirmDelete({
            heading: intl.formatMessage({
                id: 'CardRelations.deleteConfirm.heading',
                defaultMessage: 'Delete Relation',
            }),
            subText: intl.formatMessage(
                {
                    id: 'CardRelations.deleteConfirm.subText',
                    defaultMessage: 'Are you sure you want to remove the relation with "{cardTitle}"?',
                },
                {cardTitle},
            ),
            confirmButtonText: intl.formatMessage({
                id: 'CardRelations.deleteConfirm.button',
                defaultMessage: 'Delete',
            }),
            onConfirm: async () => {
                try {
                    await octoClient.deleteCardRelation(relation.id)
                    setRelations((prev) => prev.filter((r) => r.id !== relation.id))
                    sendFlashMessage({content: intl.formatMessage({
                        id: 'CardRelations.flash.removed',
                        defaultMessage: 'Relation removed',
                    }), severity: 'low'})
                } catch (error) {
                    console.error('Failed to delete relation:', error)
                    sendFlashMessage({content: intl.formatMessage({
                        id: 'CardRelations.flash.removeFailed',
                        defaultMessage: 'Failed to remove relation',
                    }), severity: 'high'})
                }
                setConfirmDelete(null)
            },
            onClose: () => setConfirmDelete(null),
        })
    }, [card.id, relatedCards])

    const handleCardClick = useCallback((cardId: string) => {
        // Navigate to the related card using react-router
        const params = {...match.params, cardId}
        const newPath = generatePath(Utils.getBoardPagePath(match.path), params)
        history.push(newPath)
    }, [match, history])

    const handleRelationCreated = useCallback(() => {
        loadRelations()
        sendFlashMessage({content: intl.formatMessage({
            id: 'CardRelations.flash.created',
            defaultMessage: 'Relation created',
        }), severity: 'low'})
    }, [loadRelations, intl])

    // Always show the section, but hide add button in readonly mode
    return (
        <div className='CardRelations'>
            <div className='CardRelations__header'>
                <span className='CardRelations__title'>
                    <FormattedMessage
                        id='CardRelations.title'
                        defaultMessage='Relations'
                    />
                </span>
                {relations.length > 0 && (
                    <span className='CardRelations__count'>
                        {relations.length}
                    </span>
                )}
            </div>

            {loading ? (
                <div className='CardRelations__loading'>
                    <FormattedMessage
                        id='CardRelations.loading'
                        defaultMessage='Loading...'
                    />
                </div>
            ) : (
                <>
                    {relations.length > 0 && (
                        <div className='CardRelations__list'>
                            {relations.map((relation) => {
                                const isSource = relation.sourceCardId === card.id
                                const relatedCardId = isSource ? relation.targetCardId : relation.sourceCardId
                                const relationType = isSource ? relation.relationType : getInverseRelationType(relation.relationType)
                                const displayType = getRelationTypeDisplayName(relationType)
                                const relatedCard = relatedCards.get(relatedCardId)

                                return (
                                    <div
                                        key={relation.id}
                                        className='CardRelations__item'
                                    >
                                        <div className='CardRelations__item-type'>
                                            {displayType}
                                        </div>
                                        <div
                                            className='CardRelations__item-card'
                                            onClick={() => handleCardClick(relatedCardId)}
                                            role='button'
                                            tabIndex={0}
                                            onKeyDown={(e) => {
                                                if (e.key === 'Enter' || e.key === ' ') {
                                                    e.preventDefault()
                                                    handleCardClick(relatedCardId)
                                                }
                                            }}
                                        >
                                            <span className='CardRelations__item-icon'>
                                                {relatedCard?.icon || '📄'}
                                            </span>
                                            <span className='CardRelations__item-title'>
                                                {relatedCard?.title || relatedCardId}
                                            </span>
                                        </div>
                                        {!readonly && (
                                            <button
                                                className='CardRelations__item-delete'
                                                onClick={() => handleDeleteRelation(relation)}
                                                aria-label='Remove relation'
                                            >
                                                <i className='CompassIcon icon-close'/>
                                            </button>
                                        )}
                                    </div>
                                )
                            })}
                        </div>
                    )}

                    {relations.length === 0 && (
                        <div className='CardRelations__empty'>
                            <FormattedMessage
                                id='CardRelations.noRelations'
                                defaultMessage='No relations yet'
                            />
                        </div>
                    )}

                    {!readonly && (
                        <div className='CardRelations__add'>
                            <button
                                className='CardRelations__add-button'
                                onClick={() => setShowAddDialog(true)}
                            >
                                <i className='CompassIcon icon-plus'/>
                                <FormattedMessage
                                    id='CardRelations.add'
                                    defaultMessage='Add relation'
                                />
                            </button>
                        </div>
                    )}
                </>
            )}

            {showAddDialog && (
                <AddRelationDialog
                    card={card}
                    boardId={boardId}
                    onClose={() => setShowAddDialog(false)}
                    onRelationCreated={handleRelationCreated}
                />
            )}

            {confirmDelete && (
                <ConfirmationDialogBox
                    dialogBox={confirmDelete}
                />
            )}
        </div>
    )
}

export default CardRelations
