// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react'
import {FormattedMessage} from 'react-intl'

import {CardRelation, getRelationTypeDisplayName} from '../../blocks/cardRelation'
import octoClient from '../../octoClient'
import {Card} from '../../blocks/card'

import './cardRelations.scss'

type Props = {
    card: Card
    readonly: boolean
}

const CardRelations = (props: Props): JSX.Element => {
    const {card, readonly} = props
    const [relations, setRelations] = useState<CardRelation[]>([])
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        loadRelations()
    }, [card.id])

    const loadRelations = async () => {
        try {
            setLoading(true)
            const cardRelations = await octoClient.getCardRelations(card.id)
            setRelations(cardRelations)
        } catch (error) {
            console.error('Failed to load card relations:', error)
        } finally {
            setLoading(false)
        }
    }

    const handleDeleteRelation = async (relationId: string) => {
        if (readonly) {
            return
        }

        try {
            await octoClient.deleteCardRelation(relationId)
            setRelations(relations.filter((r) => r.id !== relationId))
        } catch (error) {
            console.error('Failed to delete relation:', error)
        }
    }

    if (loading) {
        return (
            <div className='CardRelations'>
                <div className='CardRelations__header'>
                    <FormattedMessage
                        id='CardRelations.title'
                        defaultMessage='Relations'
                    />
                </div>
                <div className='CardRelations__loading'>
                    <FormattedMessage
                        id='CardRelations.loading'
                        defaultMessage='Loading...'
                    />
                </div>
            </div>
        )
    }

    if (relations.length === 0 && readonly) {
        return <></>
    }

    return (
        <div className='CardRelations'>
            <div className='CardRelations__header'>
                <FormattedMessage
                    id='CardRelations.title'
                    defaultMessage='Relations'
                />
            </div>
            <div className='CardRelations__list'>
                {relations.map((relation) => {
                    const isSource = relation.sourceCardId === card.id
                    const relatedCardId = isSource ? relation.targetCardId : relation.sourceCardId
                    const displayType = getRelationTypeDisplayName(relation.relationType)

                    return (
                        <div
                            key={relation.id}
                            className='CardRelations__item'
                        >
                            <div className='CardRelations__item-type'>
                                {displayType}
                            </div>
                            <div className='CardRelations__item-card'>
                                {relatedCardId}
                            </div>
                            {!readonly && (
                                <button
                                    className='CardRelations__item-delete'
                                    onClick={() => handleDeleteRelation(relation.id)}
                                >
                                    <FormattedMessage
                                        id='CardRelations.delete'
                                        defaultMessage='Remove'
                                    />
                                </button>
                            )}
                        </div>
                    )
                })}
            </div>
            {!readonly && (
                <div className='CardRelations__add'>
                    <button className='CardRelations__add-button'>
                        <FormattedMessage
                            id='CardRelations.add'
                            defaultMessage='+ Add relation'
                        />
                    </button>
                </div>
            )}
        </div>
    )
}

export default CardRelations

