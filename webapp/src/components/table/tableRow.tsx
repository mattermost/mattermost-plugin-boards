// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useMemo, useCallback} from 'react'

import {Card} from '../../blocks/card'
import {Board, IPropertyTemplate} from '../../blocks/board'
import {Constants} from '../../constants'
import {useSortable} from '../../hooks/sortable'

import PropertyValueElement from '../propertyValueElement'
import IconButton from '../../widgets/buttons/iconButton'
import CompassIcon from '../../widgets/icons/compassIcon'

import {useColumnResize} from './tableColumnResizeContext'

import './tableRow.scss'

type Props = {
    board: Board
    columnWidths: Record<string, number>
    isManualSort: boolean
    groupById?: string
    visiblePropertyIds: string[]
    collapsedOptionIds: string[]
    card: Card
    isSelected: boolean
    focusOnMount: boolean
    isLastCard: boolean
    showCard: (cardId?: string) => void
    readonly: boolean
    addCard: (groupByOptionId?: string) => Promise<void>
    onClick?: (e: React.MouseEvent<HTMLDivElement>, card: Card) => void
    onDrop: (srcCard: Card, dstCard: Card) => void
}

const TableRow = (props: Props) => {
    const {board, card, isManualSort, groupById, visiblePropertyIds, collapsedOptionIds} = props

    const isGrouped = Boolean(groupById)
    const [isDragging, isOver, cardRef] = useSortable('card', card, !props.readonly && (isManualSort || isGrouped), props.onDrop)
    const columnResize = useColumnResize()

    const onClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
        props.onClick && props.onClick(e, card)
    }, [card, props.onClick])

    const visiblePropertyTemplates = useMemo(() => (
        visiblePropertyIds.map((id) => board.cardProperties.find((t) => t.id === id)).filter((i) => i) as IPropertyTemplate[]
    ), [board.cardProperties, visiblePropertyIds])

    let className = props.isSelected ? 'TableRow octo-table-row selected' : 'TableRow octo-table-row'
    if (isOver) {
        className += ' dragover'
    }
    if (isGrouped) {
        const groupID = groupById || ''
        let groupValue = card.fields.properties[groupID] as string || 'undefined'
        if (groupValue === 'undefined') {
            const template = board.cardProperties.find((p) => p.id === groupById) //templates.find((o) => o.id === groupById)
            if (template && template.type === 'createdBy') {
                groupValue = card.createdBy
            } else if (template && template.type === 'updatedBy') {
                groupValue = card.modifiedBy
            }
        } else if (Array.isArray(groupValue)) {
            groupValue = groupValue[0]
        }
        if (collapsedOptionIds.indexOf(groupValue) > -1) {
            className += ' hidden'
        }
    }
    if (props.readonly) {
        className += ' readonly'
    }



    return (
        <div
            className={className}
            onClick={onClick}
            ref={cardRef}
            style={{opacity: isDragging ? 0.5 : 1}}
        >

            <div className='action-cell octo-table-cell-btn'>
                {!props.readonly && (
                    <IconButton icon={<CompassIcon icon='drag-vertical'/>}/>
                )}
            </div>

            {/* Code */}
            <div
                className='octo-table-cell code-cell clickable'
                style={{width: columnResize.width('code'), cursor: 'pointer'}}
                ref={(ref) => columnResize.updateRef(card.id, 'code', ref)}
                onClick={() => props.showCard(card.id)}
            >
                {card.code && <div className='card-code'>{card.code}</div>}
            </div>

            {/* Name / title */}
            <div
                className='octo-table-cell title-cell clickable'
                id='mainBoardHeader'
                style={{width: columnResize.width(Constants.titleColumnId), cursor: 'pointer'}}
                ref={(ref) => columnResize.updateRef(card.id, Constants.titleColumnId, ref)}
                onClick={() => props.showCard(card.id)}
            >
                <div className='octo-icontitle'>
                    <div className='octo-titletext'>
                        {card.title || 'Untitled'}
                    </div>
                </div>
            </div>

            {/* Columns, one per property */}
            {visiblePropertyTemplates.map((template) => {
                return (
                    <div
                        className='octo-table-cell'
                        key={template.id}
                        style={{width: columnResize.width(template.id)}}
                        ref={(ref) => columnResize.updateRef(card.id, template.id, ref)}
                    >
                        <PropertyValueElement
                            readOnly={props.readonly}
                            card={card}
                            board={board}
                            propertyTemplate={template}
                            showEmptyPlaceholder={false}
                        />
                    </div>
                )
            })}
        </div>
    )
}

export default React.memo(TableRow)
