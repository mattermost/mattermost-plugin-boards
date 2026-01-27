// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React, {useState, useCallback, useEffect, useRef} from 'react'
import {useIntl} from 'react-intl'

import {IPropertyOption} from '../../blocks/board'

import Label from '../../widgets/label'
import {Utils, IDType} from '../../utils'
import mutator from '../../mutator'
import ValueSelector from '../../widgets/valueSelector'
import octoClient from '../../octoClient'

import {PropertyProps} from '../types'

type StatusTransitionRule = {
    id: string
    boardId: string
    fromStatus: string
    toStatus: string
    allowed: boolean
    createAt: number
    updateAt: number
}

const SelectProperty = (props: PropertyProps) => {
    const {propertyValue, propertyTemplate, board, card} = props
    const intl = useIntl()

    const [open, setOpen] = useState(false)
    const [filteredOptions, setFilteredOptions] = useState<IPropertyOption[]>(propertyTemplate.options)
    const isEditable = !props.readOnly && Boolean(board)
    const currentRequestRef = useRef<string>('')

    // Check if this is a Status property (case-insensitive)
    const isStatusProperty = propertyTemplate.name.toLowerCase() === 'status'

    // Load status transition rules when the dropdown opens for Status properties
    useEffect(() => {
        if (!open || !isStatusProperty || !board || !isEditable) {
            setFilteredOptions(propertyTemplate.options)
            return
        }

        const loadTransitionRules = async () => {
            // Set this request as the current one to prevent race conditions
            const requestId = `${board.id}-${propertyValue}-${Date.now()}`
            currentRequestRef.current = requestId

            try {
                const rules = await octoClient.getStatusTransitionRules(board.id) as StatusTransitionRule[]

                // Only update state if this is still the current request
                if (currentRequestRef.current !== requestId) {
                    return
                }

                // If no rules exist, show all options (backward compatibility)
                if (!rules || rules.length === 0) {
                    setFilteredOptions(propertyTemplate.options)
                    return
                }

                // Get current status value
                const currentStatusId = propertyValue as string

                // If there's no current status, allow all statuses (initial selection)
                if (!currentStatusId) {
                    setFilteredOptions(propertyTemplate.options)
                    return
                }

                // Filter options based on transition rules
                const allowedOptions = propertyTemplate.options.filter((option) => {
                    // Always allow keeping the current status
                    if (option.id === currentStatusId) {
                        return true
                    }

                    // Check if transition from current status to this option is allowed
                    const rule = rules.find(
                        (r) => r.fromStatus === currentStatusId && r.toStatus === option.id
                    )

                    // If no rule found, allow by default (backward compatibility)
                    if (!rule) {
                        return true
                    }

                    return rule.allowed
                })

                setFilteredOptions(allowedOptions)
            } catch (error) {
                // Only update state if this is still the current request
                if (currentRequestRef.current !== requestId) {
                    return
                }
                // On error, show all options (fail-safe)
                Utils.logError(`Failed to load status transition rules: ${error}`)
                setFilteredOptions(propertyTemplate.options)
            }
        }

        loadTransitionRules()
    }, [open, isStatusProperty, board, propertyValue, propertyTemplate.options])

    const onCreate = useCallback((newValue) => {
        const option: IPropertyOption = {
            id: Utils.createGuid(IDType.BlockID),
            value: newValue,
            color: 'propColorDefault',
        }
        mutator.insertPropertyOption(board.id, board.cardProperties, propertyTemplate, option, 'add property option').then(() => {
            mutator.changePropertyValue(board.id, card, propertyTemplate.id, option.id)
        })
    }, [board, board.id, props.card, propertyTemplate.id])

    const emptyDisplayValue = props.showEmptyPlaceholder ? intl.formatMessage({id: 'PropertyValueElement.empty', defaultMessage: 'Empty'}) : ''

    const onChange = useCallback((newValue) => mutator.changePropertyValue(board.id, card, propertyTemplate.id, newValue), [board.id, card, propertyTemplate])
    const onChangeColor = useCallback((option: IPropertyOption, colorId: string) => mutator.changePropertyOptionColor(board.id, board.cardProperties, propertyTemplate, option, colorId), [board, propertyTemplate])
    const onDeleteOption = useCallback((option: IPropertyOption) => mutator.deletePropertyOption(board.id, board.cardProperties, propertyTemplate, option), [board, propertyTemplate])
    const onDeleteValue = useCallback(() => mutator.changePropertyValue(board.id, card, propertyTemplate.id, ''), [card, propertyTemplate.id])

    const option = propertyTemplate.options.find((o: IPropertyOption) => o.id === propertyValue)
    const propertyColorCssClassName = option?.color || ''
    const displayValue = option?.value
    const finalDisplayValue = displayValue || emptyDisplayValue

    if (!isEditable || !open) {
        return (
            <div
                className={props.property.valueClassName(!isEditable)}
                data-testid='select-non-editable'
                tabIndex={0}
                onClick={() => setOpen(true)}
            >
                <Label color={displayValue ? propertyColorCssClassName : 'empty'}>
                    <span className='Label-text'>{finalDisplayValue}</span>
                </Label>
            </div>
        )
    }
    return (
        <ValueSelector
            emptyValue={emptyDisplayValue}
            options={filteredOptions}
            value={propertyTemplate.options.find((p: IPropertyOption) => p.id === propertyValue)}
            onCreate={onCreate}
            onChange={onChange}
            onChangeColor={onChangeColor}
            onDeleteOption={onDeleteOption}
            onDeleteValue={onDeleteValue}
            onBlur={() => setOpen(false)}
        />
    )
}

export default React.memo(SelectProperty)
