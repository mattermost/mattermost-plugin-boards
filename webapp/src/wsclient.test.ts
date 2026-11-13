// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ACTION_SUBSCRIBE_TEAM, MMWebSocketClient, WSClient} from './wsclient'

const PLUGIN_ID = 'focalboard'
const SUBSCRIBE_TEAM_ACTION = `custom_${PLUGIN_ID}_${ACTION_SUBSCRIBE_TEAM}`

type FakeConn = {readyState: number}

// FakeMMWebSocketClient mimics the parts of the Mattermost webapp
// WebSocketClient that Boards relies on: listeners for first connect and
// reconnect, and a sendMessage that silently drops messages unless the
// current socket is open.
class FakeMMWebSocketClient implements MMWebSocketClient {
    conn: WebSocket | null = null
    sent: Array<{action: string, data: any}> = [] // eslint-disable-line @typescript-eslint/no-explicit-any
    private firstConnectListeners: Array<() => void> = []
    private reconnectListeners: Array<() => void> = []

    sendMessage(action: string, data: any): void { // eslint-disable-line @typescript-eslint/no-explicit-any
        if (this.conn && this.conn.readyState === WebSocket.OPEN) {
            this.sent.push({action, data})
        }
    }

    addFirstConnectListener(callback: () => void): void {
        this.firstConnectListeners.push(callback)
    }

    addReconnectListener(callback: () => void): void {
        this.reconnectListeners.push(callback)
    }

    addErrorListener(): void {
        // not needed for these tests
    }

    addCloseListener(): void {
        // not needed for these tests
    }

    // startConnecting replaces the socket with a new one that is still
    // connecting, like WebSocketClient.initialize() does.
    startConnecting(): FakeConn {
        const conn: FakeConn = {readyState: WebSocket.CONNECTING}
        this.conn = conn as unknown as WebSocket
        return conn
    }

    // openAsFirstConnect opens the current socket and fires the first
    // connect listeners. The webapp does this on page load and also after
    // WebSocketClient.close() + initialize(), which happens when the user
    // switches between Channels and Boards, because close() resets the
    // connection fail count.
    openAsFirstConnect(conn: FakeConn): void {
        conn.readyState = WebSocket.OPEN
        this.firstConnectListeners.forEach((listener) => listener())
    }

    // openAsReconnect opens the current socket and fires the reconnect
    // listeners, as the webapp does after a dropped connection.
    openAsReconnect(conn: FakeConn): void {
        conn.readyState = WebSocket.OPEN
        this.reconnectListeners.forEach((listener) => listener())
    }

    subscribeTeamCommands(): string[] {
        return this.sent.
            filter((msg) => msg.action === SUBSCRIBE_TEAM_ACTION).
            map((msg) => msg.data.teamId)
    }
}

describe('WSClient in plugin mode', () => {
    let mmClient: FakeMMWebSocketClient
    let client: WSClient

    beforeEach(() => {
        mmClient = new FakeMMWebSocketClient()
        client = new WSClient()
        client.initPlugin(PLUGIN_ID, '10.1.0', mmClient)
    })

    test('subscribes once when the connection is already open on open()', () => {
        mmClient.openAsFirstConnect(mmClient.startConnecting())
        client.subscribeToTeam('team-1')
        mmClient.sent = []

        client.open()

        expect(mmClient.subscribeTeamCommands()).toEqual(['team-1'])
        expect(client.state).toBe('open')
    })

    test('does not subscribe twice for the same connection', () => {
        const conn = mmClient.startConnecting()
        client.subscribeToTeam('team-1')
        client.open()

        mmClient.openAsFirstConnect(conn)

        expect(mmClient.subscribeTeamCommands()).toEqual(['team-1'])
    })

    test('resubscribes after a reconnect', () => {
        mmClient.openAsFirstConnect(mmClient.startConnecting())
        client.subscribeToTeam('team-1')
        client.open()
        mmClient.sent = []

        mmClient.openAsReconnect(mmClient.startConnecting())

        expect(mmClient.subscribeTeamCommands()).toEqual(['team-1'])
    })

    test('resubscribes when a new connection fires first connect again', () => {
        // initial page load: socket open, Boards subscribes to the team
        mmClient.openAsFirstConnect(mmClient.startConnecting())
        client.subscribeToTeam('team-1')
        client.open()
        expect(mmClient.subscribeTeamCommands()).toContain('team-1')
        mmClient.sent = []

        // user switches to Channels and back to Boards: the webapp closes
        // the socket and initializes a new one, which reports a first
        // connect instead of a reconnect
        mmClient.openAsFirstConnect(mmClient.startConnecting())

        expect(mmClient.subscribeTeamCommands()).toEqual(['team-1'])
    })

    test('sends subscriptions registered while the new socket was still connecting', () => {
        mmClient.openAsFirstConnect(mmClient.startConnecting())
        client.open()

        // route switch: the board mounts and subscribes while the new
        // socket is still connecting, so the command is dropped
        const conn = mmClient.startConnecting()
        client.subscribeToTeam('team-1')
        expect(mmClient.subscribeTeamCommands()).toEqual([])

        mmClient.openAsFirstConnect(conn)

        expect(mmClient.subscribeTeamCommands()).toEqual(['team-1'])
    })
})
