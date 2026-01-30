// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback, useEffect} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {IPropertyTemplate, IPropertyOption, PropertySortRule} from '../../blocks/board'
import Editable from '../../widgets/editable'
import IconButton from '../../widgets/buttons/iconButton'
import DeleteIcon from '../../widgets/icons/delete'
import UpIcon from '../../widgets/icons/sortUp'
import DownIcon from '../../widgets/icons/sortDown'
import ExpandIcon from '../../widgets/icons/chevronDown'
import CollapseIcon from '../../widgets/icons/chevronRight'
import propsRegistry from '../../properties'
import Menu from '../../widgets/menu'
import MenuWrapper from '../../widgets/menuWrapper'

import PropertyOptionsEditor from './propertyOptionsEditor'

import './propertyItem.scss'

type Props = {
    property: IPropertyTemplate
    index: number
    totalCount: number
    isExpanded: boolean
    onToggleExpand: () => void
    onUpdate: (property: IPropertyTemplate) => void
    onDelete: (propertyId: string) => void
    onReorder: (propertyId: string, newIndex: number) => void
}

const PropertyItem = (props: Props): JSX.Element => {
    const {property, index, totalCount, isExpanded} = props
    const intl = useIntl()
    const [name, setName] = useState(property.name)
    const [sortRule, setSortRule] = useState<PropertySortRule>(property.sortRule || 'default')

    useEffect(() => {
        setName(property.name)
        setSortRule(property.sortRule || 'default')
    }, [property.name, property.sortRule])

    const handleNameSave = useCallback(() => {
        if (name !== property.name) {
            props.onUpdate({...property, name})
        }
    }, [name, property, props])

    const handleSortRuleChange = useCallback((newSortRule: PropertySortRule) => {
        setSortRule(newSortRule)
        props.onUpdate({...property, sortRule: newSortRule})
    }, [property, props])

    const handleOptionsUpdate = useCallback((options: IPropertyOption[]) => {
        props.onUpdate({...property, options})
    }, [property, props])

    const propertyType = propsRegistry.get(property.type)
    const hasOptions = property.type === 'select' || property.type === 'multiSelect'

    const sortRuleOptions = [
        {id: 'default', name: intl.formatMessage({id: 'PropertyItem.sortRule.default', defaultMessage: 'Default (alphabetical)'})},
        {id: 'byOrder', name: intl.formatMessage({id: 'PropertyItem.sortRule.byOrder', defaultMessage: 'By order'})},
        {id: 'byValue', name: intl.formatMessage({id: 'PropertyItem.sortRule.byValue', defaultMessage: 'By value'})},
        {id: 'asNumber', name: intl.formatMessage({id: 'PropertyItem.sortRule.asNumber', defaultMessage: 'As number'})},
    ]

    const currentSortRuleName = sortRuleOptions.find((opt) => opt.id === sortRule)?.name || sortRuleOptions[0].name

    return (
        <div className='PropertyItem'>
            <div className='PropertyItem__header'>
                <IconButton
                    icon={isExpanded ? <ExpandIcon/> : <CollapseIcon/>}
                    onClick={props.onToggleExpand}
                />
                <div className='PropertyItem__name'>
                    <Editable
                        value={name}
                        placeholderText={intl.formatMessage({
                            id: 'PropertyItem.name-placeholder',
                            defaultMessage: 'Property name',
                        })}
                        onChange={setName}
                        onSave={handleNameSave}
                        saveOnEsc={true}
                    />
                </div>
                <div className='PropertyItem__type'>
                    {propertyType.displayName(intl)}
                </div>
                <div className='PropertyItem__actions'>
                    {index > 0 && (
                        <IconButton
                            icon={<UpIcon/>}
                            onClick={() => props.onReorder(property.id, index - 1)}
                        />
                    )}
                    {index < totalCount - 1 && (
                        <IconButton
                            icon={<DownIcon/>}
                            onClick={() => props.onReorder(property.id, index + 1)}
                        />
                    )}
                    <IconButton
                        icon={<DeleteIcon/>}
                        onClick={() => props.onDelete(property.id)}
                    />
                </div>
            </div>

            {isExpanded && (
                <div className='PropertyItem__details'>
                    <div className='PropertyItem__field'>
                        <label className='PropertyItem__label'>
                            <FormattedMessage
                                id='PropertyItem.sortRule'
                                defaultMessage='Sort Rule'
                            />
                        </label>
                        <MenuWrapper>
                            <button className='PropertyItem__dropdown'>
                                {currentSortRuleName}
                            </button>
                            <Menu>
                                {sortRuleOptions.map((option) => (
                                    <Menu.Text
                                        key={option.id}
                                        id={option.id}
                                        name={option.name}
                                        onClick={() => handleSortRuleChange(option.id as PropertySortRule)}
                                    />
                                ))}
                            </Menu>
                        </MenuWrapper>
                    </div>

                    {hasOptions && (
                        <PropertyOptionsEditor
                            property={property}
                            onUpdate={handleOptionsUpdate}
                        />
                    )}
                </div>
            )}
        </div>
    )
}

export default PropertyItem

