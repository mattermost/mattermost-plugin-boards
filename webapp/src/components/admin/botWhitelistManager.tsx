// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect} from 'react'

import octoClient from '../../octoClient'
import {IUser} from '../../user'

import './botWhitelistManager.scss'

type Props = {
    id: string
    label: string
    helpText?: string
    value: string[]
    disabled: boolean
    config: any
    license: any
    setByEnv: boolean
    onChange: (id: string, value: string[]) => void
    setSaveNeeded: () => void
    registerSaveAction: (action: () => Promise<void>) => void
    unRegisterSaveAction: (action: () => Promise<void>) => void
}

const BotWhitelistManager = (props: Props) => {
    const [bots, setBots] = useState<IUser[]>([])
    const [selectedBotIDs, setSelectedBotIDs] = useState<string[]>(props.value || [])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')

    useEffect(() => {
        loadBots()
    }, [])

    useEffect(() => {
        setSelectedBotIDs(props.value || [])
    }, [props.value])

    const loadBots = async () => {
        try {
            setLoading(true)
            setError('')
            
            // Get all users including bots
            const allUsers = await octoClient.getTeamUsers(false)
            
            // Filter to only bots
            const botUsers = allUsers.filter((user: IUser) => user.is_bot)
            
            setBots(botUsers)
        } catch (err) {
            setError('Failed to load bots. Please try again.')
            console.error('Error loading bots:', err)
        } finally {
            setLoading(false)
        }
    }

    const handleToggleBot = (botID: string) => {
        let newSelectedBotIDs: string[]
        
        if (selectedBotIDs.includes(botID)) {
            newSelectedBotIDs = selectedBotIDs.filter(id => id !== botID)
        } else {
            newSelectedBotIDs = [...selectedBotIDs, botID]
        }
        
        setSelectedBotIDs(newSelectedBotIDs)
        props.onChange(props.id, newSelectedBotIDs)
        props.setSaveNeeded()
    }

    const handleSelectAll = () => {
        const allBotIDs = bots.map(bot => bot.id)
        setSelectedBotIDs(allBotIDs)
        props.onChange(props.id, allBotIDs)
        props.setSaveNeeded()
    }

    const handleClearAll = () => {
        setSelectedBotIDs([])
        props.onChange(props.id, [])
        props.setSaveNeeded()
    }

    return (
        <div className='BotWhitelistManager'>
            <div className='BotWhitelistManager__header'>
                <label className='BotWhitelistManager__label'>{props.label}</label>
                {props.helpText && <p className='BotWhitelistManager__help-text'>{props.helpText}</p>}
            </div>

            {error && (
                <div className='BotWhitelistManager__error'>
                    {error}
                </div>
            )}

            {loading ? (
                <div className='BotWhitelistManager__loading'>Loading bots...</div>
            ) : (
                <div className='BotWhitelistManager__content'>
                    <div className='BotWhitelistManager__actions'>
                        <button
                            type='button'
                            onClick={handleSelectAll}
                            disabled={props.disabled || bots.length === 0}
                            className='BotWhitelistManager__button BotWhitelistManager__button--secondary'
                        >
                            Select All
                        </button>
                        <button
                            type='button'
                            onClick={handleClearAll}
                            disabled={props.disabled || selectedBotIDs.length === 0}
                            className='BotWhitelistManager__button BotWhitelistManager__button--secondary'
                        >
                            Clear All
                        </button>
                        <span className='BotWhitelistManager__count'>
                            {selectedBotIDs.length} of {bots.length} selected
                        </span>
                    </div>

                    {bots.length === 0 ? (
                        <div className='BotWhitelistManager__empty'>
                            No bots found in this team.
                        </div>
                    ) : (
                        <div className='BotWhitelistManager__list'>
                            {bots.map((bot) => (
                                <label
                                    key={bot.id}
                                    className='BotWhitelistManager__bot-item'
                                >
                                    <input
                                        type='checkbox'
                                        checked={selectedBotIDs.includes(bot.id)}
                                        onChange={() => handleToggleBot(bot.id)}
                                        disabled={props.disabled}
                                    />
                                    <span className='BotWhitelistManager__bot-name'>
                                        {bot.username}
                                    </span>
                                </label>
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    )
}

export default BotWhitelistManager

