// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {CardRelation, getRelationTypeDisplayName, getInverseRelationType} from '../../blocks/cardRelation'
import {Card} from '../../blocks/card'
import {IPropertyTemplate, IPropertyOption} from '../../blocks/board'
import {IUser} from '../../user'
import octoClient from '../../octoClient'
import {sendFlashMessage} from '../flashMessages'
import ConfirmationDialogBox, {ConfirmationDialogBoxProps} from '../confirmationDialogBox'
import {useAppSelector} from '../../store/hooks'
import {getBoard} from '../../store/boards'
import {getBoardUsers} from '../../store/users'
import {Utils} from '../../utils'
import Label from '../../widgets/label'

import AddRelationDialog from './addRelationDialog'

import './cardRelations.scss'

type Props = {
    card: Card
    boardId: string
    readonly: boolean
    showCard?: (cardId?: string) => void
}

type RelatedCardInfo = {
    id: string
    title: string
    icon: string
    code?: string
    properties: Record<string, string | string[]>
}

const CardRelations = (props: Props): JSX.Element => {
    const {card, boardId, readonly, showCard} = props
    const intl = useIntl()
    const [relations, setRelations] = useState<CardRelation[]>([])
    const [relatedCards, setRelatedCards] = useState<Map<string, RelatedCardInfo>>(new Map())
    const [loading, setLoading] = useState(true)
    const [showAddDialog, setShowAddDialog] = useState(false)
    const [confirmDelete, setConfirmDelete] = useState<ConfirmationDialogBoxProps | null>(null)

    // Get board data and users from Redux store
    const board = useAppSelector(getBoard(boardId))
    const boardUsers = useAppSelector<{[key: string]: IUser}>(getBoardUsers)

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
                            code: c.code,
                            properties: c.fields.properties || {},
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
        if (showCard) {
            showCard(cardId)
        } else {
            // Fallback: update URL directly when showCard prop is not provided
            const currentUrl = new URL(window.location.href)
            const pathParts = currentUrl.pathname.split('/')
            // Replace last segment (cardId) or append if not present
            if (pathParts.length > 0) {
                pathParts[pathParts.length - 1] = cardId
                currentUrl.pathname = pathParts.join('/')
            }
            window.history.pushState({}, '', currentUrl.toString())
            window.dispatchEvent(new PopStateEvent('popstate'))
        }
    }, [showCard])

    const handleRelationCreated = useCallback(() => {
        loadRelations()
        sendFlashMessage({content: intl.formatMessage({
            id: 'CardRelations.flash.created',
            defaultMessage: 'Relation created',
        }), severity: 'low'})
    }, [loadRelations, intl])

    // Helper function to get status property info
    // Uses type-based lookup with name as secondary filter for disambiguation
    const getStatusInfo = useCallback((relatedCard: RelatedCardInfo): {value: string, color: string} | null => {
        if (!board) {
            return null
        }

        // Find all select-type properties (potential status fields)
        const selectProperties = board.cardProperties.filter((prop: IPropertyTemplate) =>
            prop.type === 'select'
        )

        if (selectProperties.length === 0) {
            return null
        }

        // Prefer property named "Status" if multiple select properties exist,
        // otherwise use the first select property (commonly the default status field)
        let statusProperty = selectProperties.find((prop: IPropertyTemplate) =>
            prop.name.toLowerCase() === 'status'
        )
        if (!statusProperty) {
            statusProperty = selectProperties[0]
        }

        // Get the status value from the card's properties
        const statusValue = relatedCard.properties[statusProperty.id]
        if (!statusValue || typeof statusValue !== 'string') {
            return null
        }

        // Find the option that matches the status value
        const statusOption = statusProperty.options.find((opt: IPropertyOption) => opt.id === statusValue)
        if (!statusOption) {
            return null
        }

        return {
            value: statusOption.value,
            color: statusOption.color || 'propColorDefault',
        }
    }, [board])

    // Helper function to get assignee info
    // Uses type-based lookup (person/multiPerson) with name as secondary filter
    const getAssigneeInfo = useCallback((relatedCard: RelatedCardInfo): IUser | null => {
        if (!board) {
            return null
        }

        // Find all person-type properties (potential assignee fields)
        const personProperties = board.cardProperties.filter((prop: IPropertyTemplate) =>
            prop.type === 'person' || prop.type === 'multiPerson'
        )

        if (personProperties.length === 0) {
            return null
        }

        // Prefer property named "Assignee" or "Assigned to" if multiple person properties exist,
        // otherwise use the first person property
        let assigneeProperty = personProperties.find((prop: IPropertyTemplate) =>
            prop.name.toLowerCase() === 'assignee' || prop.name.toLowerCase() === 'assigned to'
        )
        if (!assigneeProperty) {
            assigneeProperty = personProperties[0]
        }

        // Get the assignee value from the card's properties
        const assigneeValue = relatedCard.properties[assigneeProperty.id]
        if (!assigneeValue) {
            return null
        }

        // Handle both single person and multi-person properties
        const userId = Array.isArray(assigneeValue) ? assigneeValue[0] : assigneeValue
        if (!userId || typeof userId !== 'string') {
            return null
        }

        return boardUsers[userId] || null
    }, [board, boardUsers])

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

                                // Get status and assignee info for the related card
                                const statusInfo = relatedCard ? getStatusInfo(relatedCard) : null
                                const assigneeInfo = relatedCard ? getAssigneeInfo(relatedCard) : null

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
                                            {relatedCard?.code && (
                                                <span className='CardRelations__item-code'>
                                                    {relatedCard.code}
                                                </span>
                                            )}
                                            <span className='CardRelations__item-title'>
                                                {relatedCard?.title || relatedCardId}
                                            </span>
                                        </div>
                                        {statusInfo && (
                                            <div className='CardRelations__item-status'>
                                                <Label color={statusInfo.color}>
                                                    <span className='Label-text'>{statusInfo.value}</span>
                                                </Label>
                                            </div>
                                        )}
                                        {assigneeInfo && (
                                            <div className='CardRelations__item-assignee'>
                                                <img
                                                    src={Utils.getProfilePicture(assigneeInfo.id)}
                                                    alt={Utils.getUserDisplayName(assigneeInfo, 'username')}
                                                    title={Utils.getUserDisplayName(assigneeInfo, 'username')}
                                                    className='CardRelations__item-avatar'
                                                />
                                            </div>
                                        )}
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
