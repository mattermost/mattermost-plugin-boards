// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react'
import {injectIntl, IntlShape} from 'react-intl'
import {generatePath, useHistory, useRouteMatch} from 'react-router-dom'

import {Board} from '../blocks/board'
import {BoardView, createBoardView, IViewType} from '../blocks/boardView'
import {Permission} from '../constants'
import mutator from '../mutator'
import TelemetryClient, {TelemetryActions, TelemetryCategory} from '../telemetry/telemetryClient'
import {Block} from '../blocks/block'
import {IDType, Utils} from '../utils'
import AddIcon from '../widgets/icons/add'
import BoardIcon from '../widgets/icons/board'
import CalendarIcon from '../widgets/icons/calendar'
import DeleteIcon from '../widgets/icons/delete'
import DuplicateIcon from '../widgets/icons/duplicate'
import GalleryIcon from '../widgets/icons/gallery'
import TableIcon from '../widgets/icons/table'
import Menu from '../widgets/menu'
import {useAppSelector} from '../store/hooks'
import {getMe} from '../store/users'

import BoardPermissionGate from './permissions/boardPermissionGate'
import './viewMenu.scss'

type Props = {
    board: Board
    activeView: BoardView
    views: BoardView[]
    intl: IntlShape
    readonly: boolean
    onRequestAddView?: (viewType: IViewType) => void
}

const ViewMenu = (props: Props) => {
    const history = useHistory()
    const match = useRouteMatch()
    const me = useAppSelector(getMe)

    const showView = useCallback((viewId) => {
        let newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, viewId: viewId || ''})
        if (props.readonly) {
            newPath += `?r=${Utils.getReadToken()}`
        }
        history.push(newPath)
    }, [match, history])

    const handleDuplicateView = useCallback(() => {
        const {board, activeView} = props
        Utils.log('duplicateView')

        TelemetryClient.trackEvent(TelemetryCategory, TelemetryActions.DuplicateBoardView, {board: board.id, view: activeView.id})
        const currentViewId = activeView.id
        const newView = createBoardView(activeView)
        newView.title = `${activeView.title} copy`
        newView.id = Utils.createGuid(IDType.View)
        mutator.insertBlock(
            newView.boardId,
            newView,
            'duplicate view',
            async (block: Block) => {
                // This delay is needed because WSClient has a default 100 ms notification delay before updates
                setTimeout(() => {
                    showView(block.id)
                }, 120)
            },
            async () => {
                showView(currentViewId)
            },
        )
    }, [props.activeView, showView])

    const handleDeleteView = useCallback(() => {
        const {board, activeView, views} = props
        Utils.log('deleteView')
        TelemetryClient.trackEvent(TelemetryCategory, TelemetryActions.DeleteBoardView, {board: board.id, view: activeView.id})
        const view = activeView
        const nextView = views.find((o) => o.id !== view.id)
        mutator.deleteBlock(view, 'delete view')
        if (nextView) {
            showView(nextView.id)
        }
    }, [props.views, props.activeView, showView])

    const handleViewClick = useCallback((id: string) => {
        const {views} = props
        Utils.log('view ' + id)
        const view = views.find((o) => o.id === id)
        Utils.assert(view, `view not found: ${id}`)
        if (view) {
            showView(view.id)
        }
    }, [props.views, showView])

    const handleAddViewBoard = useCallback(() => {
        props.onRequestAddView?.('board')
    }, [props.onRequestAddView])

    const handleAddViewTable = useCallback(() => {
        props.onRequestAddView?.('table')
    }, [props.onRequestAddView])

    const handleAddViewGallery = useCallback(() => {
        props.onRequestAddView?.('gallery')
    }, [props.onRequestAddView])

    const handleAddViewCalendar = useCallback(() => {
        props.onRequestAddView?.('calendar')
    }, [props.onRequestAddView])

    const {views, intl} = props

    const duplicateViewText = intl.formatMessage({
        id: 'View.DuplicateView',
        defaultMessage: 'Duplicate view',
    })
    const deleteViewText = intl.formatMessage({
        id: 'View.DeleteView',
        defaultMessage: 'Delete view',
    })
    const addViewText = intl.formatMessage({
        id: 'View.AddView',
        defaultMessage: 'Add view',
    })
    const boardText = intl.formatMessage({
        id: 'View.Board',
        defaultMessage: 'Board',
    })
    const tableText = intl.formatMessage({
        id: 'View.Table',
        defaultMessage: 'Table',
    })
    const galleryText = intl.formatMessage({
        id: 'View.Gallery',
        defaultMessage: 'Gallery',
    })

    const iconForViewType = (viewType: IViewType) => {
        switch (viewType) {
        case 'board': return <BoardIcon/>
        case 'table': return <TableIcon/>
        case 'gallery': return <GalleryIcon/>
        case 'calendar': return <CalendarIcon/>
        default: return <div/>
        }
    }

    return (
        <div className='ViewMenu'>
            <Menu>
                <div className='view-list'>
                    {views.map((view: BoardView) => (
                        <Menu.Text
                            key={view.id}
                            id={view.id}
                            name={view.title}
                            icon={iconForViewType(view.fields.viewType)}
                            onClick={handleViewClick}
                        />))}
                </div>
                <BoardPermissionGate permissions={[Permission.ManageBoardProperties]}>
                    <Menu.Separator/>
                </BoardPermissionGate>
                {!props.readonly &&
                <BoardPermissionGate permissions={[Permission.ManageBoardProperties]}>
                    <Menu.Text
                        id='__duplicateView'
                        name={duplicateViewText}
                        icon={<DuplicateIcon/>}
                        onClick={handleDuplicateView}
                    />
                </BoardPermissionGate>
                }
                {!props.readonly && views.length > 1 && me && (props.activeView.fields.ownerUserId || props.activeView.createdBy) === me.id &&
                <BoardPermissionGate permissions={[Permission.ManageBoardProperties]}>
                    <Menu.Text
                        id='__deleteView'
                        name={deleteViewText}
                        icon={<DeleteIcon/>}
                        onClick={handleDeleteView}
                    />
                </BoardPermissionGate>
                }
                {!props.readonly &&
                <BoardPermissionGate permissions={[Permission.ManageBoardProperties]}>
                    <Menu.SubMenu
                        id='__addView'
                        name={addViewText}
                        icon={<AddIcon/>}
                    >
                        <div className='subMenu'>
                            <Menu.Text
                                id='board'
                                name={boardText}
                                icon={<BoardIcon/>}
                                onClick={handleAddViewBoard}
                            />
                            <Menu.Text
                                id='table'
                                name={tableText}
                                icon={<TableIcon/>}
                                onClick={handleAddViewTable}
                            />
                            <Menu.Text
                                id='gallery'
                                name={galleryText}
                                icon={<GalleryIcon/>}
                                onClick={handleAddViewGallery}
                            />
                            <Menu.Text
                                id='calendar'
                                name='Calendar'
                                icon={<CalendarIcon/>}
                                onClick={handleAddViewCalendar}
                            />
                        </div>
                    </Menu.SubMenu>
                </BoardPermissionGate>
                }
            </Menu>
        </div>
    )
}

export default injectIntl(React.memo(ViewMenu))
