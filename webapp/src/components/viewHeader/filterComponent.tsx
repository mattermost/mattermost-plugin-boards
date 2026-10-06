// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {FormattedMessage} from 'react-intl'

import {FilterClause, FilterCondition, createFilterClause} from '../../blocks/filterClause'
import {createFilterGroup, isAFilterGroupInstance} from '../../blocks/filterGroup'
import {Board, IPropertyTemplate} from '../../blocks/board'
import {BoardView} from '../../blocks/boardView'
import mutator from '../../mutator'
import {Utils} from '../../utils'
import {OctoUtils} from '../../octoUtils'
import Button from '../../widgets/buttons/button'
import propsRegistry from '../../properties'
import {FilterValueType} from '../../properties/types'

import Modal from '../modal'

import FilterEntry from './filterEntry'

import './filterComponent.scss'

type Props = {
    board: Board
    activeView: BoardView
    onClose: () => void
}

// Title is always a valid filter target (see filterEntry's property menu and
// CardFilter.isClauseMet), so it is offered once every card property is in use.
const titlePropertyId = 'title'

// Returns the clause that "+ Add filter" should create next, or undefined when
// every filterable property (and the title) is already being filtered on.
export function nextFilterClause(board: Board, filters: FilterClause[]): FilterClause | undefined {
    const usedPropertyIds = new Set(filters.map((f) => f.propertyId))

    const property = board.cardProperties.
        filter((o: IPropertyTemplate) => propsRegistry.get(o.type).canFilter).
        find((o: IPropertyTemplate) => !usedPropertyIds.has(o.id))

    let propertyId: string
    let filterValueType: FilterValueType
    if (property) {
        propertyId = property.id
        filterValueType = propsRegistry.get(property.type).filterValueType
    } else if (!usedPropertyIds.has(titlePropertyId)) {
        propertyId = titlePropertyId
        filterValueType = 'text'
    } else {
        return undefined
    }

    const filter = createFilterClause()
    filter.propertyId = propertyId
    filter.condition = OctoUtils.filterConditionValidOrDefault(filterValueType, filter.condition)
    return filter
}

const FilterComponent = (props: Props): React.JSX.Element => {
    const {board, activeView} = props

    // Ignore extra clicks until the view includes the last added clause.
    // changeViewFilter PATCHes over the network; the view updates later.
    const pendingPropertyIdRef = React.useRef<string | undefined>(undefined)
    const [pendingPropertyId, setPendingPropertyId] = React.useState<string | undefined>(undefined)

    const filters: FilterClause[] = activeView.fields.filter?.filters.filter((o) => !isAFilterGroupInstance(o)) as FilterClause[] || []
    const filterPropertyKey = filters.map((f) => f.propertyId).join(',')

    React.useEffect(() => {
        if (!pendingPropertyId) {
            return
        }
        if (filters.some((f) => f.propertyId === pendingPropertyId)) {
            pendingPropertyIdRef.current = undefined
            setPendingPropertyId(undefined)
        }
    }, [filterPropertyKey, pendingPropertyId, filters])

    const conditionClicked = (optionId: string, filter: FilterClause): void => {
        const filterIndex = activeView.fields.filter.filters.indexOf(filter)
        Utils.assert(filterIndex >= 0, "Can't find filter")

        const filterGroup = createFilterGroup(activeView.fields.filter)
        const newFilter = filterGroup.filters[filterIndex] as FilterClause

        Utils.assert(newFilter, `No filter at index ${filterIndex}`)
        if (newFilter.condition !== optionId) {
            newFilter.condition = optionId as FilterCondition
            mutator.changeViewFilter(board.id, activeView.id, activeView.fields.filter, filterGroup)
        }
    }

    const addFilterClicked = () => {
        // Read the live view so a completed update is visible on the next
        // click, while an in-flight PATCH (unchanged view) is ignored.
        const currentFilters = activeView.fields.filter?.filters.filter((o) => !isAFilterGroupInstance(o)) as FilterClause[] || []
        if (pendingPropertyIdRef.current && !currentFilters.some((f) => f.propertyId === pendingPropertyIdRef.current)) {
            return
        }

        const filter = nextFilterClause(board, currentFilters)
        if (!filter) {
            return
        }

        pendingPropertyIdRef.current = filter.propertyId
        setPendingPropertyId(filter.propertyId)

        const filterGroup = createFilterGroup(activeView.fields.filter)
        filterGroup.filters.push(filter)

        Promise.resolve(mutator.changeViewFilter(board.id, activeView.id, activeView.fields.filter, filterGroup)).catch(() => {
            pendingPropertyIdRef.current = undefined
            setPendingPropertyId(undefined)
        })
    }

    const awaitingAdd = Boolean(pendingPropertyId && !filters.some((f) => f.propertyId === pendingPropertyId))
    const canAddFilter = Boolean(nextFilterClause(board, filters)) && !awaitingAdd

    return (
        <Modal
            onClose={props.onClose}
        >
            <div
                className='FilterComponent'
            >
                {filters.map((filter) => (
                    <FilterEntry
                        key={`${filter.propertyId}-${filter.condition}`}
                        board={board}
                        view={activeView}
                        conditionClicked={conditionClicked}
                        filter={filter}
                    />
                ))}

                <br/>

                <Button
                    onClick={() => addFilterClicked()}
                    disabled={!canAddFilter}
                    className={canAddFilter ? '' : 'disabled'}
                >
                    <FormattedMessage
                        id='FilterComponent.add-filter'
                        defaultMessage='+ Add filter'
                    />
                </Button>
            </div>
        </Modal>
    )
}

export default React.memo(FilterComponent)
