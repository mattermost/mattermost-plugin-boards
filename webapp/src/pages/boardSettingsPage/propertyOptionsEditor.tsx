// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {IPropertyTemplate, IPropertyOption} from '../../blocks/board'
import Editable from '../../widgets/editable'
import IconButton from '../../widgets/buttons/iconButton'
import DeleteIcon from '../../widgets/icons/delete'
import UpIcon from '../../widgets/icons/sortUp'
import DownIcon from '../../widgets/icons/sortDown'
import Button from '../../widgets/buttons/button'
import {Utils, IDType} from '../../utils'

import './propertyOptionsEditor.scss'

type Props = {
    property: IPropertyTemplate
    onUpdate: (options: IPropertyOption[]) => void
}

const PropertyOptionsEditor = (props: Props): JSX.Element => {
    const {property} = props
    const intl = useIntl()

    const handleAddOption = useCallback(() => {
        const newOption: IPropertyOption = {
            id: Utils.createGuid(IDType.BlockID),
            value: intl.formatMessage({id: 'PropertyOptionsEditor.new-option', defaultMessage: 'New option'}),
            color: 'propColorDefault',
            hideIfEmpty: false,
        }
        props.onUpdate([...property.options, newOption])
    }, [property.options, props, intl])

    const handleUpdateOption = useCallback((optionId: string, updates: Partial<IPropertyOption>) => {
        const updatedOptions = property.options.map((opt) =>
            opt.id === optionId ? {...opt, ...updates} : opt
        )
        props.onUpdate(updatedOptions)
    }, [property.options, props])

    const handleDeleteOption = useCallback((optionId: string) => {
        const updatedOptions = property.options.filter((opt) => opt.id !== optionId)
        props.onUpdate(updatedOptions)
    }, [property.options, props])

    const handleReorderOption = useCallback((optionId: string, newIndex: number) => {
        const currentIndex = property.options.findIndex((opt) => opt.id === optionId)
        if (currentIndex === -1 || currentIndex === newIndex) {
            return
        }

        const newOptions = [...property.options]
        const [movedOption] = newOptions.splice(currentIndex, 1)
        newOptions.splice(newIndex, 0, movedOption)

        props.onUpdate(newOptions)
    }, [property.options, props])

    const colorOptions = [
        'propColorDefault',
        'propColorGray',
        'propColorBrown',
        'propColorOrange',
        'propColorYellow',
        'propColorGreen',
        'propColorBlue',
        'propColorPurple',
        'propColorPink',
        'propColorRed',
    ]

    return (
        <div className='PropertyOptionsEditor'>
            <div className='PropertyOptionsEditor__header'>
                <FormattedMessage
                    id='PropertyOptionsEditor.title'
                    defaultMessage='Options'
                />
            </div>

            <div className='PropertyOptionsEditor__list'>
                {property.options.map((option, index) => (
                    <div
                        key={option.id}
                        className='PropertyOptionsEditor__option'
                    >
                        <div className='PropertyOptionsEditor__option-color'>
                            <select
                                value={option.color}
                                onChange={(e) => handleUpdateOption(option.id, {color: e.target.value})}
                                className={`PropertyOptionsEditor__color-select ${option.color}`}
                            >
                                {colorOptions.map((color) => (
                                    <option
                                        key={color}
                                        value={color}
                                    >
                                        {color.replace('propColor', '')}
                                    </option>
                                ))}
                            </select>
                        </div>

                        <div className='PropertyOptionsEditor__option-value'>
                            <Editable
                                value={option.value}
                                placeholderText={intl.formatMessage({
                                    id: 'PropertyOptionsEditor.option-placeholder',
                                    defaultMessage: 'Option value',
                                })}
                                onChange={(newValue) => handleUpdateOption(option.id, {value: newValue})}
                                saveOnEsc={true}
                            />
                        </div>

                        <div className='PropertyOptionsEditor__option-hide'>
                            <label>
                                <input
                                    type='checkbox'
                                    checked={option.hideIfEmpty || false}
                                    onChange={(e) => handleUpdateOption(option.id, {hideIfEmpty: e.target.checked})}
                                />
                                <FormattedMessage
                                    id='PropertyOptionsEditor.hide-if-empty'
                                    defaultMessage='Hide if empty'
                                />
                            </label>
                        </div>

                        <div className='PropertyOptionsEditor__option-actions'>
                            {index > 0 && (
                                <IconButton
                                    icon={<UpIcon/>}
                                    onClick={() => handleReorderOption(option.id, index - 1)}
                                />
                            )}
                            {index < property.options.length - 1 && (
                                <IconButton
                                    icon={<DownIcon/>}
                                    onClick={() => handleReorderOption(option.id, index + 1)}
                                />
                            )}
                            <IconButton
                                icon={<DeleteIcon/>}
                                onClick={() => handleDeleteOption(option.id)}
                            />
                        </div>
                    </div>
                ))}
            </div>

            <div className='PropertyOptionsEditor__add'>
                <Button onClick={handleAddOption}>
                    <FormattedMessage
                        id='PropertyOptionsEditor.add-option'
                        defaultMessage='+ Add Option'
                    />
                </Button>
            </div>
        </div>
    )
}

export default PropertyOptionsEditor

