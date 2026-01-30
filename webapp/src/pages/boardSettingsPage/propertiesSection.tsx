// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Board, IPropertyTemplate, PropertyTypeEnum} from '../../blocks/board'
import Button from '../../widgets/buttons/button'
import Menu from '../../widgets/menu'
import MenuWrapper from '../../widgets/menuWrapper'
import {Utils, IDType} from '../../utils'
import propsRegistry from '../../properties'
import {PropertyTypes} from '../../widgets/propertyMenu'
import {GITHUB_PRS_PROPERTY_ID} from '../../components/cardDetail/githubPRStatus'

import PropertyItem from './propertyItem'

import './propertiesSection.scss'

type Props = {
    board: Board
    onBoardChange: (board: Board) => Promise<void> | void
}

const PropertiesSection = (props: Props): JSX.Element => {
    const {board} = props
    const intl = useIntl()
    const [expandedPropertyId, setExpandedPropertyId] = useState<string | null>(null)

    const handleAddProperty = useCallback((type: string) => {
        const propertyType = propsRegistry.get(type as PropertyTypeEnum)
        const newProperty: IPropertyTemplate = {
            id: Utils.createGuid(IDType.BlockID),
            name: propertyType.displayName(intl),
            type: type as PropertyTypeEnum,
            options: [],
            sortRule: 'default',
        }

        const updatedBoard = {
            ...board,
            cardProperties: [...board.cardProperties, newProperty],
        }

        props.onBoardChange(updatedBoard)
        setExpandedPropertyId(newProperty.id)
    }, [board, props, intl])

    const handleUpdateProperty = useCallback((updatedProperty: IPropertyTemplate) => {
        const updatedBoard = {
            ...board,
            cardProperties: board.cardProperties.map((prop) =>
                prop.id === updatedProperty.id ? updatedProperty : prop
            ),
        }
        props.onBoardChange(updatedBoard)
    }, [board, props])

    const handleDeleteProperty = useCallback((propertyId: string) => {
        const updatedBoard = {
            ...board,
            cardProperties: board.cardProperties.filter((prop) => prop.id !== propertyId),
        }
        props.onBoardChange(updatedBoard)
    }, [board, props])

    const handleReorderProperty = useCallback((propertyId: string, newIndex: number) => {
        const currentIndex = board.cardProperties.findIndex((prop) => prop.id === propertyId)
        if (currentIndex === -1 || currentIndex === newIndex) {
            return
        }

        const newProperties = [...board.cardProperties]
        const [movedProperty] = newProperties.splice(currentIndex, 1)
        newProperties.splice(newIndex, 0, movedProperty)

        const updatedBoard = {
            ...board,
            cardProperties: newProperties,
        }
        props.onBoardChange(updatedBoard)
    }, [board, props])

    // Issue 9: Filter out GitHub PRs property (it's force-hidden and managed by external cron)
    const visibleProperties = board.cardProperties.filter((prop) => prop.id !== GITHUB_PRS_PROPERTY_ID)

    return (
        <div className='PropertiesSection'>
            <div className='PropertiesSection__list'>
                {visibleProperties.map((property, index) => (
                    <PropertyItem
                        key={property.id}
                        property={property}
                        index={index}
                        totalCount={visibleProperties.length}
                        isExpanded={expandedPropertyId === property.id}
                        onToggleExpand={() => setExpandedPropertyId(
                            expandedPropertyId === property.id ? null : property.id
                        )}
                        onUpdate={handleUpdateProperty}
                        onDelete={handleDeleteProperty}
                        onReorder={handleReorderProperty}
                    />
                ))}
            </div>

            <div className='PropertiesSection__add'>
                <MenuWrapper>
                    <Button>
                        <FormattedMessage
                            id='PropertiesSection.add-property'
                            defaultMessage='+ Add Property'
                        />
                    </Button>
                    <Menu>
                        <PropertyTypes
                            label={intl.formatMessage({
                                id: 'PropertiesSection.select-type',
                                defaultMessage: 'Select property type',
                            })}
                            onTypeSelected={(type) => handleAddProperty(type.type)}
                        />
                    </Menu>
                </MenuWrapper>
            </div>
        </div>
    )
}

export default PropertiesSection

