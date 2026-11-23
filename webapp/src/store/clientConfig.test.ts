// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {configureStore} from '@reduxjs/toolkit'

import {getLegacyMode, getMovedBoardsReadOnly, reducer, setClientConfig} from './clientConfig'

import {RootState} from './index'

const makeStore = () => configureStore({reducer: {clientConfig: reducer}})
const stateOf = (store: ReturnType<typeof makeStore>) => store.getState() as unknown as RootState

describe('store/clientConfig', () => {
    test('moved boards are read-only by default, but only once Legacy Mode is on', () => {
        const store = makeStore()

        expect(stateOf(store).clientConfig.value.movedBoardsReadOnly).toBe(true)
        expect(getLegacyMode(stateOf(store))).toBe(false)
        expect(getMovedBoardsReadOnly(stateOf(store))).toBe(false)
    })

    test('getMovedBoardsReadOnly needs both Legacy Mode and the setting', () => {
        const store = makeStore()
        const base = stateOf(store).clientConfig.value

        store.dispatch(setClientConfig({...base, legacyMode: true, movedBoardsReadOnly: true}))
        expect(getMovedBoardsReadOnly(stateOf(store))).toBe(true)

        store.dispatch(setClientConfig({...base, legacyMode: true, movedBoardsReadOnly: false}))
        expect(getMovedBoardsReadOnly(stateOf(store))).toBe(false)

        store.dispatch(setClientConfig({...base, legacyMode: false, movedBoardsReadOnly: true}))
        expect(getMovedBoardsReadOnly(stateOf(store))).toBe(false)
    })
})
