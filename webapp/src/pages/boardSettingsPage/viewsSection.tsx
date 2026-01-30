// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'
import {ActionMeta} from 'react-select'

import {Board} from '../../blocks/board'
import {BoardView, IViewType} from '../../blocks/boardView'
import {IUser} from '../../user'
import mutator from '../../mutator'
import Menu from '../../widgets/menu'
import MenuWrapper from '../../widgets/menuWrapper'
import Button from '../../widgets/buttons/button'
import PersonSelector from '../../components/personSelector'
import {useAppDispatch} from '../../store/hooks'
import {updateViews} from '../../store/views'

import './viewsSection.scss'

type Props = {
    board: Board
    views: BoardView[]
    boardUsers: IUser[]
}

const VIEW_TYPE_OPTIONS: {value: IViewType, labelId: string, defaultLabel: string}[] = [
    {value: 'board', labelId: 'ViewType.board', defaultLabel: 'Board'},
    {value: 'table', labelId: 'ViewType.table', defaultLabel: 'Table'},
    {value: 'gallery', labelId: 'ViewType.gallery', defaultLabel: 'Gallery'},
    {value: 'calendar', labelId: 'ViewType.calendar', defaultLabel: 'Calendar'},
]

const ViewsSection = (props: Props): JSX.Element => {
    const {views, boardUsers} = props
    const intl = useIntl()
    const dispatch = useAppDispatch()

    const viewTypeLabels = VIEW_TYPE_OPTIONS.map((opt) => ({
        ...opt,
        label: intl.formatMessage({id: opt.labelId, defaultMessage: opt.defaultLabel}),
    }))

    const handleViewTypeChange = useCallback(async (view: BoardView, newType: IViewType) => {
        if (view.fields.viewType === newType) {
            return
        }

        const updatedView = {...view, fields: {...view.fields, viewType: newType}}
        await mutator.updateBlock(
            view.boardId,
            updatedView,
            view,
            'change view type',
        )
        // Update Redux store immediately (Issue 5)
        dispatch(updateViews([updatedView]))
    }, [dispatch])

    const handleOwnerChange = useCallback(async (view: BoardView, newOwner: IUser | null, action: ActionMeta<IUser>) => {
        if (action.action === 'clear') {
            // Don't allow clearing the owner
            return
        }

        const newOwnerId = newOwner?.id
        const currentOwnerId = view.fields.ownerUserId || view.createdBy
        if (!newOwnerId || currentOwnerId === newOwnerId) {
            return
        }

        // Store owner in fields.ownerUserId so it persists via the
        // BlockPatch updatedFields (createdBy is a read-only top-level
        // field that the patch API ignores).
        const updatedView = {...view, fields: {...view.fields, ownerUserId: newOwnerId}}
        await mutator.updateBlock(
            view.boardId,
            updatedView,
            view,
            'change view owner',
        )
        // Update Redux store immediately
        dispatch(updateViews([updatedView]))
    }, [dispatch])

    const handleVisibilityChange = useCallback(async (view: BoardView, isOwnerOnly: boolean) => {
        const currentVisibility = view.fields.visibility || 'everyone'
        const newVisibility = isOwnerOnly ? 'owner-only' : 'everyone'

        if (currentVisibility === newVisibility) {
            return
        }

        const updatedView = {...view, fields: {...view.fields, visibility: newVisibility}}
        await mutator.updateBlock(
            view.boardId,
            updatedView,
            view,
            'change view visibility',
        )
        // Update Redux store immediately (Issue 4)
        dispatch(updateViews([updatedView]))
    }, [dispatch])

    const getOwnerUser = useCallback((view: BoardView): IUser | undefined => {
        const ownerId = view.fields.ownerUserId || view.createdBy
        return boardUsers.find(u => u.id === ownerId)
    }, [boardUsers])

    return (
        <div className='ViewsSection'>
            <div className='ViewsSection__table'>
                <div className='ViewsSection__table-header'>
                    <div className='ViewsSection__table-cell ViewsSection__table-cell--name'>
                        <FormattedMessage
                            id='ViewsSection.header.name'
                            defaultMessage='View Name'
                        />
                    </div>
                    <div className='ViewsSection__table-cell ViewsSection__table-cell--type'>
                        <FormattedMessage
                            id='ViewsSection.header.type'
                            defaultMessage='Type'
                        />
                    </div>
                    <div className='ViewsSection__table-cell ViewsSection__table-cell--owner'>
                        <FormattedMessage
                            id='ViewsSection.header.owner'
                            defaultMessage='Owner'
                        />
                    </div>
                    <div className='ViewsSection__table-cell ViewsSection__table-cell--visibility'>
                        <FormattedMessage
                            id='ViewsSection.header.visibility'
                            defaultMessage='Visibility'
                        />
                    </div>
                </div>

                {views.map((view) => {
                    const owner = getOwnerUser(view)
                    const visibility = view.fields.visibility || 'everyone'
                    const isOwnerOnly = visibility === 'owner-only'

                    return (
                        <div
                            key={view.id}
                            className='ViewsSection__table-row'
                        >
                            <div className='ViewsSection__table-cell ViewsSection__table-cell--name'>
                                {view.title || intl.formatMessage({id: 'ViewsSection.untitled', defaultMessage: 'Untitled'})}
                            </div>
                            <div className='ViewsSection__table-cell ViewsSection__table-cell--type'>
                                <MenuWrapper>
                                    <Button>
                                        {viewTypeLabels.find(opt => opt.value === view.fields.viewType)?.label ||
                                            intl.formatMessage({id: 'ViewType.board', defaultMessage: 'Board'})}
                                    </Button>
                                    <Menu>
                                        {viewTypeLabels.map((option) => (
                                            <Menu.Text
                                                key={option.value}
                                                id={option.value}
                                                name={option.label}
                                                onClick={() => handleViewTypeChange(view, option.value)}
                                            />
                                        ))}
                                    </Menu>
                                </MenuWrapper>
                            </div>
                            <div className='ViewsSection__table-cell ViewsSection__table-cell--owner'>
                                <PersonSelector
                                    readOnly={false}
                                    userIDs={owner ? [owner.id] : []}
                                    allowAddUsers={false}
                                    emptyDisplayValue={intl.formatMessage({id: 'ViewsSection.no-owner', defaultMessage: 'No owner'})}
                                    isMulti={false}
                                    closeMenuOnSelect={true}
                                    onChange={(user: IUser | null, action: ActionMeta<IUser>) => handleOwnerChange(view, user, action)}
                                />
                            </div>
                            <div className='ViewsSection__table-cell ViewsSection__table-cell--visibility'>
                                <MenuWrapper>
                                    <Button>
                                        {isOwnerOnly ?
                                            intl.formatMessage({id: 'ViewsSection.visibility.owner-only', defaultMessage: 'Owner only'}) :
                                            intl.formatMessage({id: 'ViewsSection.visibility.everyone', defaultMessage: 'Everyone'})
                                        }
                                    </Button>
                                    <Menu>
                                        <Menu.Text
                                            id='everyone'
                                            name={intl.formatMessage({id: 'ViewsSection.visibility.everyone', defaultMessage: 'Everyone'})}
                                            onClick={() => handleVisibilityChange(view, false)}
                                        />
                                        <Menu.Text
                                            id='owner-only'
                                            name={intl.formatMessage({id: 'ViewsSection.visibility.owner-only', defaultMessage: 'Owner only'})}
                                            onClick={() => handleVisibilityChange(view, true)}
                                        />
                                    </Menu>
                                </MenuWrapper>
                            </div>
                        </div>
                    )
                })}
            </div>
        </div>
    )
}

export default React.memo(ViewsSection)
