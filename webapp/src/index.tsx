/* eslint-disable */
// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
import {InternalPluginRegistry, PluginRegistry} from '@hmhealey/plugin-support'
import {WebSocketMessage} from '@mattermost/client'
import {PostEmbed, PostEmbedType} from '@mattermost/types/posts'
import {GlobalState} from '@mattermost/types/store'
import React, {useEffect} from 'react'
import {createIntl, createIntlCache} from 'react-intl'
import {Store, Action} from 'redux'
import {Provider as ReduxProvider} from 'react-redux'
import {createBrowserHistory, History} from 'history'
import {rudderAnalytics, RudderTelemetryHandler} from 'mattermost-redux/client/rudder'
import {selectTeam} from 'mattermost-redux/actions/teams'

import appBarIcon from '../static/app-bar-icon.png'

import TelemetryClient, {TelemetryActions, TelemetryCategory} from './telemetry/telemetryClient'
import {setMattermostTheme} from './theme'
import FocalboardIcon from './widgets/icons/logo'
import GlobalHeader from './components/globalHeader/globalHeader'
import App from './app'
import store from './store'
import {setTeam} from './store/teams'
import WithWebSockets from './components/withWebSockets'
import {setChannel} from './store/channels'
import {initialLoad} from './store/initialLoad'
import {Utils} from './utils'
import './styles/focalboard-variables.scss'
import './styles/main.scss'
import './styles/labels.scss'
import octoClient from './octoClient'
import {Board} from './blocks/board'
import {getMessages, getCurrentLanguage} from './i18n'
import {UserSettings} from './userSettings'
import {SuiteWindow} from './types/index'
import BoardsUnfurl from './components/boardsUnfurl/boardsUnfurl'
import RHSChannelBoards from './components/rhsChannelBoards'
import RHSChannelBoardsHeader from './components/rhsChannelBoardsHeader'
import BoardSelector from './components/boardSelector'
import wsClient, {
    ACTION_UPDATE_BLOCK,
    ACTION_UPDATE_CLIENT_CONFIG,
    ACTION_UPDATE_SUBSCRIPTION,
    ACTION_UPDATE_CARD_LIMIT_TIMESTAMP,
    ACTION_UPDATE_CATEGORY,
    ACTION_UPDATE_BOARD_CATEGORY,
    ACTION_UPDATE_BOARD,
    ACTION_REORDER_CATEGORIES,
} from './wsclient'
import manifest from './manifest'
import ErrorBoundary from './error_boundary'
// eslint-disable-next-line import/no-unresolved
import './plugin.scss'
import CloudUpgradeNudge from "./components/cloudUpgradeNudge/cloudUpgradeNudge"
import CreateBoardFromTemplate from './components/createBoardFromTemplate'

const windowAny = (window as SuiteWindow)
windowAny.baseURL = process.env.TARGET_IS_PRODUCT ? '/plugins/boards' : '/plugins/focalboard'
windowAny.frontendBaseURL = '/boards'
windowAny.isFocalboardPlugin = true

function getSubpath(siteURL: string): string {
    const url = new URL(siteURL)

    // remove trailing slashes
    return url.pathname.replace(/\/+$/, '')
}

const TELEMETRY_RUDDER_KEY = 'placeholder_rudder_key'
const TELEMETRY_RUDDER_DATAPLANE_URL = 'placeholder_rudder_dataplane_url'
const TELEMETRY_OPTIONS = {
    context: {
        ip: '0.0.0.0',
    },
    page: {
        path: '',
        referrer: '',
        search: '',
        title: '',
        url: '',
    },
    anonymousId: '00000000000000000000000000',
}

function customHistory() {
    const history = createBrowserHistory({basename: Utils.getFrontendBaseURL()})

    if (Utils.isDesktop()) {
        window.addEventListener('message', (event: MessageEvent) => {
            if (event.origin !== windowAny.location.origin) {
                return
            }

            const pathName = event.data.message?.pathName
            if (!pathName || !pathName.startsWith('/boards')) {
                return
            }

            Utils.log(`Navigating Boards to ${pathName}`)
            history.replace(pathName.replace('/boards', ''))
        })
    }
    return {
        ...history,
        push: (path: string, state?: unknown) => {
            if (Utils.isDesktop()) {
                windowAny.postMessage(
                    {
                        type: 'browser-history-push',
                        message: {
                            path: `${windowAny.frontendBaseURL}${path}`,
                        },
                    },
                    windowAny.location.origin,
                )
            } else {
                history.push(path, state as Record<string, never>)
            }
        },
    }
}

let browserHistory: History<unknown>

const MainApp = () => {
    useEffect(() => {
        document.body.classList.add('focalboard-body')
        document.body.classList.add('app__body')
        const root = document.getElementById('root')
        if (root) {
            root.classList.add('focalboard-plugin-root')
        }

        return () => {
            document.body.classList.remove('focalboard-body')
            document.body.classList.remove('app__body')
            if (root) {
                root.classList.remove('focalboard-plugin-root')
            }
        }
    }, [])

    return (
        <ErrorBoundary>
            <ReduxProvider store={store}>
                <WithWebSockets manifest={manifest}>
                    <div id='focalboard-app'>
                        <App history={browserHistory} />
                    </div>
                    <div id='focalboard-root-portal' />
                </WithWebSockets>
            </ReduxProvider>
        </ErrorBoundary>
    )
}

const HeaderComponent = () => {
    return (
        <ErrorBoundary>
            <GlobalHeader history={browserHistory} />
        </ErrorBoundary>
    )
}

export default class Plugin {
    channelHeaderButtonId?: string
    rhsId?: string
    boardSelectorId?: string
    registry?: PluginRegistry & InternalPluginRegistry

    async initialize(registry: PluginRegistry & InternalPluginRegistry, mmStore: Store<GlobalState, Action<Record<string, unknown>>>): Promise<void> {
        const siteURL = mmStore.getState().entities.general.config.SiteURL
        const subpath = siteURL ? getSubpath(siteURL) : ''
        windowAny.frontendBaseURL = subpath + windowAny.frontendBaseURL
        windowAny.baseURL = subpath + windowAny.baseURL
        browserHistory = customHistory()
        const cache = createIntlCache()
        const intl = createIntl({
            // modeled after <IntlProvider> in webapp/src/app.tsx
            locale: getCurrentLanguage(),
            messages: getMessages(getCurrentLanguage())
        }, cache)


        this.registry = registry

        UserSettings.nameFormat = mmStore.getState().entities.preferences?.myPreferences['display_settings--name_format']?.value || null
        let theme = mmStore.getState().entities.preferences.myPreferences.theme
        setMattermostTheme(theme)

        const productID = process.env.TARGET_IS_PRODUCT ? 'boards' : manifest.id

        // register websocket handlers
        this.registry?.registerWebSocketEventHandler({
            event: `custom_${productID}_${ACTION_UPDATE_BOARD}`,
            handler: (e: WebSocketMessage) => wsClient.updateHandler(e.data)})
        this.registry?.registerWebSocketEventHandler({
            event: `custom_${productID}_${ACTION_UPDATE_CATEGORY}`,
            handler: (e: WebSocketMessage) => wsClient.updateHandler(e.data)})
        this.registry?.registerWebSocketEventHandler({
            event: `custom_${productID}_${ACTION_UPDATE_BOARD_CATEGORY}`,
            handler: (e: WebSocketMessage) => wsClient.updateHandler(e.data)
        })
        this.registry?.registerWebSocketEventHandler({
            event: `custom_${productID}_${ACTION_UPDATE_CLIENT_CONFIG}`,
            handler: (e: WebSocketMessage) => wsClient.updateClientConfigHandler(e.data)
        })
        this.registry?.registerWebSocketEventHandler({
            event: `custom_${productID}_${ACTION_UPDATE_CARD_LIMIT_TIMESTAMP}`,
            handler: (e: WebSocketMessage) => wsClient.updateCardLimitTimestampHandler(e.data)
        })
        this.registry?.registerWebSocketEventHandler({
            event: `custom_${productID}_${ACTION_UPDATE_SUBSCRIPTION}`,
            handler: (e: WebSocketMessage) => wsClient.updateSubscriptionHandler(e.data)
        })
        this.registry?.registerWebSocketEventHandler({
            event: `custom_${productID}_${ACTION_REORDER_CATEGORIES}`,
            handler: (e: WebSocketMessage) => wsClient.updateHandler(e.data)
        })

        this.registry?.registerWebSocketEventHandler({
            event: 'plugin_statuses_changed',
            handler: (e: WebSocketMessage) => wsClient.pluginStatusesChangedHandler(e.data)
        })
        this.registry?.registerPostTypeComponent({
            type: 'custom_cloud_upgrade_nudge',
            component: CloudUpgradeNudge
        })
        this.registry?.registerWebSocketEventHandler({
            event: 'preferences_changed',
            handler: (e: WebSocketMessage) => {
                let preferences
                try {
                    preferences = JSON.parse(e.data.preferences)
                } catch {
                    preferences = []
                }
                if (preferences) {
                    for (const preference of preferences) {
                        if (preference.category === 'theme' && theme !== preference.value) {
                            setMattermostTheme(JSON.parse(preference.value))
                            theme = preference.value
                        }
                        if (preference.category === 'display_settings' && preference.name === 'name_format') {
                            UserSettings.nameFormat = preference.value
                        }
                    }
                }
            }
        })

        let lastViewedChannel = mmStore.getState().entities.channels.currentChannelId
        let prevTeamID: string

        const currentChannel = mmStore.getState().entities.channels.currentChannelId
        const currentChannelObj = mmStore.getState().entities.channels.channels[currentChannel]
        store.dispatch(setChannel(currentChannelObj))

        mmStore.subscribe(() => {
            const currentUserId = mmStore.getState().entities.users.currentUserId
            const currentChannel = mmStore.getState().entities.channels.currentChannelId
            if (lastViewedChannel !== currentChannel && currentChannel) {
                localStorage.setItem('focalboardLastViewedChannel:' + currentUserId, currentChannel)
                lastViewedChannel = currentChannel
                octoClient.channelId = currentChannel
                const currentChannelObj = mmStore.getState().entities.channels.channels[lastViewedChannel]
                store.dispatch(setChannel(currentChannelObj))
            }

            // Watch for change in active team.
            // This handles the user selecting a team from the team sidebar.
            const currentTeamID = mmStore.getState().entities.teams.currentTeamId
            if (currentTeamID && currentTeamID !== prevTeamID) {
                if (prevTeamID && window.location.pathname.startsWith(windowAny.frontendBaseURL || '')) {
                    // Don't re-push the URL if we're already on a URL for the current team
                    if (!window.location.pathname.startsWith(`${(windowAny.frontendBaseURL || '')}/team/${currentTeamID}`))
                        browserHistory.push(`/team/${currentTeamID}`)
                }
                prevTeamID = currentTeamID
                store.dispatch(setTeam(currentTeamID))
                octoClient.teamId = currentTeamID
                store.dispatch(initialLoad())
            }

            if (currentTeamID && currentTeamID !== prevTeamID) {
                let theme = mmStore.getState().entities.preferences.myPreferences[`theme--${currentTeamID}`]
                if (!theme) {
                    theme = mmStore.getState().entities.preferences.myPreferences['theme--'] || mmStore.getState().entities.preferences.myPreferences.theme
                }
                setMattermostTheme(theme)
            }
        })

        let fbPrevTeamID = store.getState().teams.currentId
        store.subscribe(() => {
            const currentTeamID: string = store.getState().teams.currentId
            const currentUserId = mmStore.getState().entities.users.currentUserId
            if (currentTeamID !== fbPrevTeamID) {
                fbPrevTeamID = currentTeamID
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                mmStore.dispatch(selectTeam(currentTeamID))
                localStorage.setItem(`user_prev_team:${currentUserId}`, currentTeamID)
            }
        })

        if (this.registry.registerProduct) {
            windowAny.frontendBaseURL = subpath + '/boards'

            // TODO rhsId isn't a value returned by this?
            const {rhsId, toggleRHSPlugin} = this.registry.registerRightHandSidebarComponent({
                component: () => (
                    <ReduxProvider store={store}>
                        <WithWebSockets manifest={manifest}>
                            <RHSChannelBoards />
                        </WithWebSockets>
                    </ReduxProvider>
                ),
                title: <ErrorBoundary>
                    <ReduxProvider store={store}>
                        <RHSChannelBoardsHeader />
                    </ReduxProvider>
                </ErrorBoundary>
                ,
            }) as any
            this.rhsId = rhsId

            this.channelHeaderButtonId = registry.registerChannelHeaderButtonAction({
                icon: <FocalboardIcon />,
                action: () => mmStore.dispatch(toggleRHSPlugin),
                dropdownText: 'Boards',
                tooltipText: 'Boards',
            })

            this.registry.registerProduct({
                baseURL: '/boards',
                switcherIcon: 'product-boards',
                switcherText: 'Boards',
                switcherLinkURL: '/boards',
                mainComponent: MainApp,
                headerCentreComponent: HeaderComponent,
                headerRightComponent: () => null,
                showTeamSidebar: true,
            })

            if (this.registry.registerAppBarComponent) {
                this.registry.registerAppBarComponent({
                    iconUrl: Utils.buildURL(appBarIcon, true),
                    action: () => mmStore.dispatch(toggleRHSPlugin),
                    tooltipText: intl.formatMessage({id: 'AppBar.Tooltip', defaultMessage: 'Toggle Linked Boards'})
                })
            }

            if (this.registry.registerActionAfterChannelCreation) {
                this.registry.registerActionAfterChannelCreation({
                    component: (props: {
                        setCanCreate: (canCreate: boolean) => void,
                        setAction: (fn: () => (channelId: string, teamId: string) => Promise<Board | undefined>) => void,
                        newBoardInfoIcon: React.ReactNode,
                    }) => (
                        <ReduxProvider store={store}>
                            <CreateBoardFromTemplate
                                setCanCreate={props.setCanCreate}
                                setAction={props.setAction}
                                newBoardInfoIcon={props.newBoardInfoIcon}
                            />
                        </ReduxProvider>
                    )
                })
            }

            this.registry.registerPostWillRenderEmbedComponent({
                match: (embed: PostEmbed) => embed.type === 'boards' as PostEmbedType,
                component: (props: {embed: {data: string}}) => (
                    <ReduxProvider store={store}>
                        <BoardsUnfurl
                            embed={props.embed}
                        />
                    </ReduxProvider>
                ),
                toggleable: false
            })

            // Site statistics handler
            if (registry.registerSiteStatisticsHandler) {
                registry.registerSiteStatisticsHandler({
                    handler: async () => {
                        const siteStats = await octoClient.getSiteStatistics()
                        if (siteStats) {
                            return {
                                boards_count: {
                                    name: intl.formatMessage({id: 'SiteStats.total_boards', defaultMessage: 'Total Boards'}),
                                    id: 'total_boards',
                                    icon: 'icon-product-boards',
                                    value: siteStats.board_count,
                                },
                                cards_count: {
                                    name: intl.formatMessage({id: 'SiteStats.total_cards', defaultMessage: 'Total Cards'}),
                                    id: 'total_cards',
                                    icon: 'icon-products',
                                    value: siteStats.card_count,
                                },
                            }
                        }
                        return {}
                    }
                })
            }
        }

        this.boardSelectorId = this.registry.registerRootComponent({
            component: () => (
                <ReduxProvider store={store}>
                    <WithWebSockets manifest={manifest}>
                        <BoardSelector />
                    </WithWebSockets>
                </ReduxProvider>
            )
        })

        const config = await octoClient.getClientConfig()
        if (config?.telemetry) {
            let rudderKey = TELEMETRY_RUDDER_KEY
            let rudderUrl = TELEMETRY_RUDDER_DATAPLANE_URL

            if (rudderKey.startsWith('placeholder') && rudderUrl.startsWith('placeholder')) {
                rudderKey = process.env.RUDDER_KEY as string //eslint-disable-line no-process-env
                rudderUrl = process.env.RUDDER_DATAPLANE_URL as string //eslint-disable-line no-process-env
            }

            if (rudderKey !== '') {
                const rudderCfg = {} as {setCookieDomain: string}
                if (siteURL && siteURL !== '') {
                    try {
                        rudderCfg.setCookieDomain = new URL(siteURL).hostname
                        // eslint-disable-next-line no-empty
                    } catch (_) { }
                }
                rudderAnalytics.load(rudderKey, rudderUrl, rudderCfg)

                rudderAnalytics.identify(config?.telemetryid, {}, TELEMETRY_OPTIONS)

                rudderAnalytics.page('BoardsLoaded', '',
                    TELEMETRY_OPTIONS.page,
                    {
                        context: TELEMETRY_OPTIONS.context,
                        anonymousId: TELEMETRY_OPTIONS.anonymousId,
                    })

                rudderAnalytics.ready(() => {
                    TelemetryClient.setTelemetryHandler(new RudderTelemetryHandler())
                })
            }
        }

        windowAny.getCurrentTeamId = (): string => {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            return mmStore.getState().entities.teams.currentTeamId
        }
    }

    uninitialize(): void {
        if (this.channelHeaderButtonId) {
            this.registry?.unregisterComponent({componentId: this.channelHeaderButtonId})
        }
        if (this.rhsId) {
            this.registry?.unregisterComponent({componentId: this.rhsId})
        }
        if (this.boardSelectorId) {
            this.registry?.unregisterComponent({componentId: this.boardSelectorId})
        }

        // unregister websocket handlers
        this.registry?.unregisterWebSocketEventHandler({event: wsClient.clientPrefix + ACTION_UPDATE_BLOCK})
    }
}
