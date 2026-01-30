// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react'
import {FormattedMessage, useIntl} from 'react-intl'
import {generatePath, useHistory, useRouteMatch} from 'react-router-dom'

import ViewMenu from '../../components/viewMenu'
import mutator from '../../mutator'
import {Board, IPropertyTemplate} from '../../blocks/board'
import {BoardView, createBoardView, IViewType, ViewVisibility} from '../../blocks/boardView'
import {Block} from '../../blocks/block'
import {Constants} from '../../constants'
import TelemetryClient, {TelemetryActions, TelemetryCategory} from '../../telemetry/telemetryClient'
import {Utils} from '../../utils'
import {Card} from '../../blocks/card'
import Button from '../../widgets/buttons/button'
import IconButton from '../../widgets/buttons/iconButton'
import DropdownIcon from '../../widgets/icons/dropdown'
import MenuWrapper from '../../widgets/menuWrapper'
import Editable from '../../widgets/editable'

import ModalWrapper from '../modalWrapper'

import {useAppSelector} from '../../store/hooks'
import {Permission} from '../../constants'
import {useHasCurrentBoardPermissions} from '../../hooks/permissions'
import {
    getOnboardingTourCategory,
    getOnboardingTourStarted,
    getOnboardingTourStep,
} from '../../store/users'
import {
    BoardTourSteps,
    TOUR_BOARD,
    TourCategoriesMapToSteps,
} from '../onboardingTour'
import {OnboardingBoardTitle} from '../cardDetail/cardDetail'
import AddViewTourStep from '../onboardingTour/addView/add_view'
import {getCurrentCard} from '../../store/cards'
import BoardPermissionGate from '../permissions/boardPermissionGate'
import ViewVisibilityDialog from '../viewVisibilityDialog'
import RootPortal from '../rootPortal'

import NewCardButton from './newCardButton'
import ViewHeaderPropertiesMenu from './viewHeaderPropertiesMenu'
import ViewHeaderGroupByMenu from './viewHeaderGroupByMenu'
import ViewHeaderDisplayByMenu from './viewHeaderDisplayByMenu'
import ViewHeaderSortMenu from './viewHeaderSortMenu'
import ViewHeaderActionsMenu from './viewHeaderActionsMenu'
import ViewHeaderSearch from './viewHeaderSearch'
import FilterComponent from './filterComponent'

import './viewHeader.scss'

type Props = {
    board: Board
    activeView: BoardView
    views: BoardView[]
    cards: Card[]
    groupByProperty?: IPropertyTemplate
    addCard: () => void
    addCardFromTemplate: (cardTemplateId: string) => void
    addCardTemplate: () => void
    editCardTemplate: (cardTemplateId: string) => void
    readonly: boolean
    dateDisplayProperty?: IPropertyTemplate
}

const ViewHeader = (props: Props) => {
    const [showFilter, setShowFilter] = useState(false)
    const [lockFilterOnClose, setLockFilterOnClose] = useState(false)
    const [showVisibilityDialog, setShowVisibilityDialog] = useState(false)
    const [pendingViewType, setPendingViewType] = useState<IViewType | null>(null)
    const intl = useIntl()
    const history = useHistory()
    const match = useRouteMatch()
    const canEditBoardProperties = useHasCurrentBoardPermissions([Permission.ManageBoardProperties])

    const {board, activeView, views, groupByProperty, cards, dateDisplayProperty} = props

    const withGroupBy = activeView.fields.viewType === 'board' || activeView.fields.viewType === 'table'
    const withDisplayBy = activeView.fields.viewType === 'calendar'
    const withSortBy = activeView.fields.viewType !== 'calendar'

    const [viewTitle, setViewTitle] = useState(activeView.title)

    useEffect(() => {
        setViewTitle(activeView.title)
    }, [activeView.title])

    const hasFilter = activeView.fields.filter && activeView.fields.filter.filters?.length > 0

    const isOnboardingBoard = props.board.title === OnboardingBoardTitle
    const onboardingTourStarted = useAppSelector(getOnboardingTourStarted)
    const onboardingTourCategory = useAppSelector(getOnboardingTourCategory)
    const onboardingTourStep = useAppSelector(getOnboardingTourStep)

    const currentCard = useAppSelector(getCurrentCard)
    const noCardOpen = !currentCard

    const showTourBaseCondition = isOnboardingBoard &&
        onboardingTourStarted &&
        noCardOpen &&
        onboardingTourCategory === TOUR_BOARD &&
        onboardingTourStep === BoardTourSteps.ADD_VIEW.toString()

    const [delayComplete, setDelayComplete] = useState(false)

    useEffect(() => {
        if (showTourBaseCondition) {
            setTimeout(() => {
                setDelayComplete(true)
            }, 800)
        }
    }, [showTourBaseCondition])

    useEffect(() => {
        if (!BoardTourSteps.SHARE_BOARD) {
            BoardTourSteps.SHARE_BOARD = 2
        }

        TourCategoriesMapToSteps[TOUR_BOARD] = BoardTourSteps
    }, [])

    const showAddViewTourStep = showTourBaseCondition && delayComplete

    const showView = useCallback((viewId: string) => {
        let newPath = generatePath(Utils.getBoardPagePath(match.path), {...match.params, viewId: viewId || ''})
        if (props.readonly) {
            newPath += `?r=${Utils.getReadToken()}`
        }
        history.push(newPath)
    }, [match, history, props.readonly])

    const createViewWithVisibility = useCallback((viewType: IViewType, visibility: ViewVisibility) => {
        Utils.log(`addview-${viewType}`)
        TelemetryClient.trackEvent(TelemetryCategory, TelemetryActions.CreateBoardView, {board: board.id, view: activeView.id})
        const view = createBoardView()

        switch (viewType) {
        case 'board':
            view.title = intl.formatMessage({id: 'View.NewBoardTitle', defaultMessage: 'Board view'})
            break
        case 'table':
            view.title = intl.formatMessage({id: 'View.NewTableTitle', defaultMessage: 'Table view'})
            view.fields.visiblePropertyIds = board.cardProperties.map((o: IPropertyTemplate) => o.id)
            view.fields.columnWidths = {}
            view.fields.columnWidths[Constants.titleColumnId] = Constants.defaultTitleColumnWidth
            break
        case 'gallery':
            view.title = intl.formatMessage({id: 'View.NewGalleryTitle', defaultMessage: 'Gallery view'})
            view.fields.visiblePropertyIds = [Constants.titleColumnId]
            break
        case 'calendar':
            view.title = intl.formatMessage({id: 'View.NewCalendarTitle', defaultMessage: 'Calendar view'})
            view.parentId = board.id
            view.fields.visiblePropertyIds = [Constants.titleColumnId]
            view.fields.dateDisplayPropertyId = board.cardProperties.find((o: IPropertyTemplate) => o.type === 'date')?.id
            break
        }

        view.fields.viewType = viewType
        view.boardId = board.id
        view.fields.visibility = visibility === 'everyone' ? 'everyone' : 'owner-only'

        const oldViewId = activeView.id

        mutator.insertBlock(
            view.boardId,
            view,
            'add view',
            async (block: Block) => {
                setTimeout(() => {
                    showView(block.id)
                }, 120)
            },
            async () => {
                showView(oldViewId)
            },
        )
    }, [activeView, board, intl, showView])

    const handleRequestAddView = useCallback((viewType: IViewType) => {
        setPendingViewType(viewType)
        setShowVisibilityDialog(true)
    }, [])

    const handleVisibilityPublic = useCallback(() => {
        if (pendingViewType) {
            createViewWithVisibility(pendingViewType, 'everyone')
        }
        setShowVisibilityDialog(false)
        setPendingViewType(null)
    }, [pendingViewType, createViewWithVisibility])

    const handleVisibilityPersonal = useCallback(() => {
        if (pendingViewType) {
            createViewWithVisibility(pendingViewType, 'owner-only')
        }
        setShowVisibilityDialog(false)
        setPendingViewType(null)
    }, [pendingViewType, createViewWithVisibility])

    const handleVisibilityCancel = useCallback(() => {
        setShowVisibilityDialog(false)
        setPendingViewType(null)
    }, [])

    return (
        <div className='ViewHeader'>
            <div className='viewSelector'>
                <Editable
                    value={viewTitle}
                    placeholderText='Untitled View'
                    onSave={(): void => {
                        mutator.changeBlockTitle(activeView.boardId, activeView.id, activeView.title, viewTitle)
                    }}
                    onCancel={(): void => {
                        setViewTitle(activeView.title)
                    }}
                    onChange={setViewTitle}
                    saveOnEsc={true}
                    readonly={props.readonly || !canEditBoardProperties}
                    spellCheck={true}
                    autoExpand={false}
                />
                {!props.readonly && (<div>
                    <MenuWrapper label={intl.formatMessage({id: 'ViewHeader.view-menu', defaultMessage: 'View menu'})}>
                        <IconButton icon={<DropdownIcon/>}/>
                        <ViewMenu
                            board={board}
                            activeView={activeView}
                            views={views}
                            readonly={props.readonly || !canEditBoardProperties}
                            onRequestAddView={handleRequestAddView}
                        />
                    </MenuWrapper>
                    {showAddViewTourStep && <AddViewTourStep/>}
                </div>)}

            </div>

            <div className='octo-spacer'/>

            {!props.readonly && canEditBoardProperties &&
            <>
                {/* Card properties */}

                <ViewHeaderPropertiesMenu
                    properties={board.cardProperties}
                    activeView={activeView}
                />

                {/* Group by */}

                {withGroupBy &&
                <ViewHeaderGroupByMenu
                    properties={board.cardProperties}
                    activeView={activeView}
                    groupByProperty={groupByProperty}
                />}

                {/* Display by */}

                {withDisplayBy &&
                <ViewHeaderDisplayByMenu
                    properties={board.cardProperties}
                    activeView={activeView}
                    dateDisplayPropertyName={dateDisplayProperty?.name}
                />}

                {/* Filter */}

                <ModalWrapper>
                    <Button
                        active={hasFilter}
                        onClick={() => setShowFilter(!showFilter)}
                        onMouseOver={() => setLockFilterOnClose(true)}
                        onMouseLeave={() => setLockFilterOnClose(false)}
                    >
                        <FormattedMessage
                            id='ViewHeader.filter'
                            defaultMessage='Filter'
                        />
                    </Button>
                    {showFilter &&
                    <FilterComponent
                        board={board}
                        activeView={activeView}
                        onClose={() => {
                            if (!lockFilterOnClose) {
                                setShowFilter(false)
                            }
                        }}
                    />}
                </ModalWrapper>

                {/* Sort */}

                {withSortBy &&
                <ViewHeaderSortMenu
                    properties={board.cardProperties}
                    activeView={activeView}
                    orderedCards={cards}
                />
                }
            </>
            }

            {/* Search */}

            <ViewHeaderSearch/>

            {/* Options menu */}

            {!props.readonly &&
            <>
                <ViewHeaderActionsMenu
                    board={board}
                    activeView={activeView}
                    cards={cards}
                />

                {/* New card button */}

                <BoardPermissionGate permissions={[Permission.ManageBoardCards]}>
                    <NewCardButton
                        addCard={props.addCard}
                        addCardFromTemplate={props.addCardFromTemplate}
                        addCardTemplate={props.addCardTemplate}
                        editCardTemplate={props.editCardTemplate}
                    />
                </BoardPermissionGate>
            </>}
            {showVisibilityDialog &&
                <RootPortal>
                    <ViewVisibilityDialog
                        dialogBox={{
                            onPublic: handleVisibilityPublic,
                            onPersonal: handleVisibilityPersonal,
                            onClose: handleVisibilityCancel,
                        }}
                    />
                </RootPortal>
            }
        </div>
    )
}

export default React.memo(ViewHeader)
