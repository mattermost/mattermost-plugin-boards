// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {Board, IPropertyTemplate} from '../../blocks/board'
import {Card} from '../../blocks/card'
import {BoardView} from '../../blocks/boardView'
import {GITHUB_PRS_PROPERTY_ID} from './githubPRStatus'

import mutator from '../../mutator'
import Button from '../../widgets/buttons/button'
import MenuWrapper from '../../widgets/menuWrapper'
import PropertyMenu, {PropertyTypes} from '../../widgets/propertyMenu'

import Calculations from '../calculations/calculations'
import PropertyValueElement from '../propertyValueElement'
import ConfirmationDialogBox, {ConfirmationDialogBoxProps} from '../confirmationDialogBox'
import {sendFlashMessage} from '../flashMessages'
import Menu from '../../widgets/menu'
import {IDType, Utils} from '../../utils'
import AddPropertiesTourStep from '../onboardingTour/addProperties/add_properties'
import {Permission} from '../../constants'
import {useHasCurrentBoardPermissions} from '../../hooks/permissions'
import propRegistry from '../../properties'
import {PropertyType} from '../../properties/types'

type Props = {
    board: Board
    card: Card
    cards: Card[]
    activeView: BoardView
    views: BoardView[]
    readonly: boolean
}

const CardDetailProperties = (props: Props) => {
    const {board, card, cards, views, activeView} = props
    const [newTemplateId, setNewTemplateId] = useState('')
    const [showHiddenProperties, setShowHiddenProperties] = useState(false)
    const canEditBoardProperties = useHasCurrentBoardPermissions([Permission.ManageBoardProperties])
    const canEditBoardCards = useHasCurrentBoardPermissions([Permission.ManageBoardCards])
    const intl = useIntl()

    useEffect(() => {
        const newProperty = board.cardProperties.find((property) => property.id === newTemplateId)
        if (newProperty) {
            setNewTemplateId('')
        }
    }, [newTemplateId, board.cardProperties])

    const [confirmationDialogBox, setConfirmationDialogBox] = useState<ConfirmationDialogBoxProps>({heading: '', onConfirm: () => {}, onClose: () => {}})
    const [showConfirmationDialog, setShowConfirmationDialog] = useState<boolean>(false)

    function onPropertyChangeSetAndOpenConfirmationDialog(newType: PropertyType, newName: string, propertyTemplate: IPropertyTemplate) {
        const oldType = propRegistry.get(propertyTemplate.type)

        // do nothing if no change
        if (oldType === newType && propertyTemplate.name === newName) {
            return
        }

        const affectsNumOfCards: string = Calculations.countNotEmpty(cards, propertyTemplate, intl)

        // if only the name has changed, set the property without warning
        if (affectsNumOfCards === '0' || oldType === newType) {
            mutator.changePropertyTypeAndName(board, cards, propertyTemplate, newType.type, newName)
            return
        }

        const subTextString = intl.formatMessage({
            id: 'CardDetailProperty.property-name-change-subtext',
            defaultMessage: 'type from "{oldPropType}" to "{newPropType}"',
        }, {oldPropType: oldType.displayName(intl), newPropType: newType.displayName(intl)})

        setConfirmationDialogBox({
            heading: intl.formatMessage({id: 'CardDetailProperty.confirm-property-type-change', defaultMessage: 'Confirm property type change'}),
            subText: intl.formatMessage({
                id: 'CardDetailProperty.confirm-property-name-change-subtext',
                defaultMessage: 'Are you sure you want to change property "{propertyName}" {customText}? This will affect value(s) across {numOfCards} card(s) in this board, and can result in data loss.',
            },
            {
                propertyName: propertyTemplate.name,
                customText: subTextString,
                numOfCards: affectsNumOfCards,
            }),

            confirmButtonText: intl.formatMessage({id: 'CardDetailProperty.property-change-action-button', defaultMessage: 'Change property'}),
            onConfirm: async () => {
                setShowConfirmationDialog(false)
                try {
                    await mutator.changePropertyTypeAndName(board, cards, propertyTemplate, newType.type, newName)
                } catch (err: any) {
                    Utils.logError(`Error Changing Property And Name:${propertyTemplate.name}: ${err?.toString()}`)
                }
                sendFlashMessage({content: intl.formatMessage({id: 'CardDetailProperty.property-changed', defaultMessage: 'Changed property successfully!'}), severity: 'high'})
            },
            onClose: () => setShowConfirmationDialog(false),
        })

        // open confirmation dialog for property type change
        setShowConfirmationDialog(true)
    }

    function onPropertyDeleteSetAndOpenConfirmationDialog(propertyTemplate: IPropertyTemplate) {
        // set ConfirmationDialogBox Props
        setConfirmationDialogBox({
            heading: intl.formatMessage({id: 'CardDetailProperty.confirm-delete-heading', defaultMessage: 'Confirm delete property'}),
            subText: intl.formatMessage({
                id: 'CardDetailProperty.confirm-delete-subtext',
                defaultMessage: 'Are you sure you want to delete the property "{propertyName}"? Deleting it will delete the property from all cards in this board.',
            },
            {propertyName: propertyTemplate.name}),
            confirmButtonText: intl.formatMessage({id: 'CardDetailProperty.delete-action-button', defaultMessage: 'Delete'}),
            onConfirm: async () => {
                const deletingPropName = propertyTemplate.name
                setShowConfirmationDialog(false)
                try {
                    await mutator.deleteProperty(board, views, cards, propertyTemplate.id)
                    sendFlashMessage({content: intl.formatMessage({id: 'CardDetailProperty.property-deleted', defaultMessage: 'Deleted {propertyName} successfully!'}, {propertyName: deletingPropName}), severity: 'high'})
                } catch (err: any) {
                    Utils.logError(`Error Deleting Property!: Could Not delete Property -" + ${deletingPropName} ${err?.toString()}`)
                }
            },

            onClose: () => setShowConfirmationDialog(false),
        })

        // open confirmation dialog property delete
        setShowConfirmationDialog(true)
    }

    // Helper function to check if a property value is empty
    const isPropertyEmpty = (propertyTemplate: IPropertyTemplate): boolean => {
        const value = card.fields.properties[propertyTemplate.id]
        if (value === undefined || value === null || value === '') {
            return true
        }
        if (Array.isArray(value) && value.length === 0) {
            return true
        }
        return false
    }

    // Separate properties into visible and hidden based on hideIfEmpty setting
    // Issue 6: hideIfEmpty now applies to entire property, not individual options
    // Issue 9: Exclude GitHub PRs property — it's force-hidden and rendered separately by GitHubPRStatus
    const visibleProperties: IPropertyTemplate[] = []
    const hiddenProperties: IPropertyTemplate[] = []

    board.cardProperties.forEach((propertyTemplate: IPropertyTemplate) => {
        // Skip GitHub PRs property — it's rendered by GitHubPRStatus component, not as a regular property
        if (propertyTemplate.id === GITHUB_PRS_PROPERTY_ID) {
            return
        }

        const isEmpty = isPropertyEmpty(propertyTemplate)
        const shouldHide = isEmpty && propertyTemplate.hideIfEmpty

        if (shouldHide) {
            hiddenProperties.push(propertyTemplate)
        } else {
            visibleProperties.push(propertyTemplate)
        }
    })

    const renderPropertyRow = (propertyTemplate: IPropertyTemplate) => (
        <div
            key={propertyTemplate.id + '-' + propertyTemplate.type}
            className='octo-propertyrow'
        >
            {(props.readonly || !canEditBoardProperties) && <div className='octo-propertyname octo-propertyname--readonly'>{propertyTemplate.name}</div>}
            {!props.readonly && canEditBoardProperties &&
                <MenuWrapper isOpen={propertyTemplate.id === newTemplateId}>
                    <div className='octo-propertyname'><Button>{propertyTemplate.name}</Button></div>
                    <PropertyMenu
                        propertyId={propertyTemplate.id}
                        propertyName={propertyTemplate.name}
                        propertyType={propRegistry.get(propertyTemplate.type)}
                        onTypeAndNameChanged={(newType: PropertyType, newName: string) => onPropertyChangeSetAndOpenConfirmationDialog(newType, newName, propertyTemplate)}
                        onDelete={() => onPropertyDeleteSetAndOpenConfirmationDialog(propertyTemplate)}
                    />
                </MenuWrapper>
            }
            <PropertyValueElement
                readOnly={props.readonly || !canEditBoardCards}
                card={card}
                board={board}
                propertyTemplate={propertyTemplate}
                showEmptyPlaceholder={true}
            />
        </div>
    )

    return (
        <div className='octo-propertylist CardDetailProperties'>
            {visibleProperties.map(renderPropertyRow)}

            {hiddenProperties.length > 0 && (
                <>
                    <div
                        className='CardDetailProperties__display-more'
                        onClick={() => setShowHiddenProperties(!showHiddenProperties)}
                    >
                        <FormattedMessage
                            id='CardDetail.display-more'
                            defaultMessage='-- Display More --'
                        />
                    </div>
                    {showHiddenProperties && hiddenProperties.map(renderPropertyRow)}
                </>
            )}

            {showConfirmationDialog && (
                <ConfirmationDialogBox
                    dialogBox={confirmationDialogBox}
                />
            )}

            {!props.readonly && canEditBoardProperties &&
                <div className='octo-propertyname add-property'>
                    <MenuWrapper>
                        <Button>
                            <FormattedMessage
                                id='CardDetail.add-property'
                                defaultMessage='+ Add a property'
                            />
                        </Button>
                        <Menu>
                            <PropertyTypes
                                label={intl.formatMessage({id: 'PropertyMenu.selectType', defaultMessage: 'Select property type'})}
                                onTypeSelected={async (type) => {
                                    const template: IPropertyTemplate = {
                                        id: Utils.createGuid(IDType.BlockID),
                                        name: type.displayName(intl),
                                        type: type.type,
                                        options: [],
                                    }
                                    const templateId = await mutator.insertPropertyTemplate(board, activeView, -1, template)
                                    setNewTemplateId(templateId)
                                }}
                            />
                        </Menu>
                    </MenuWrapper>

                    <AddPropertiesTourStep/>
                </div>
            }
        </div>
    )
}

export default React.memo(CardDetailProperties)
