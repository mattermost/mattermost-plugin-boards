// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect} from 'react'

import './figmaTokenManager.scss'

type Props = {
    id: string
    label: string
    helpText?: string
    value: string
    disabled: boolean
    config: any
    license: any
    setByEnv: boolean
    onChange: (id: string, value: string) => void
    setSaveNeeded: () => void
    registerSaveAction: (action: () => Promise<void>) => void
    unRegisterSaveAction: (action: () => Promise<void>) => void
}

const FigmaTokenManager = (props: Props) => {
    const [isEditing, setIsEditing] = useState(false)
    const [tokenValue, setTokenValue] = useState('')
    const [displayValue, setDisplayValue] = useState('')

    useEffect(() => {
        if (props.value && props.value.length > 16) {
            const masked = props.value.substring(0, 8) + '...' + props.value.substring(props.value.length - 8)
            setDisplayValue(masked)
        } else if (props.value) {
            setDisplayValue(props.value)
        } else {
            setDisplayValue('')
        }
    }, [props.value])

    const handleEditClick = () => {
        setIsEditing(true)
        setTokenValue('')
    }

    const handleCancelClick = () => {
        setIsEditing(false)
        setTokenValue('')
    }

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = e.target.value
        setTokenValue(newValue)

        if (newValue.trim() !== '') {
            props.onChange(props.id, newValue)
            props.setSaveNeeded()
        }
    }

    return (
        <div className='FigmaTokenManager'>
            <div className='FigmaTokenManager__header'>
                <label className='FigmaTokenManager__label'>{props.label}</label>
                {props.helpText && <p className='FigmaTokenManager__help-text'>{props.helpText}</p>}
            </div>

            <div className='FigmaTokenManager__content'>
                {isEditing ? (
                    <div className='FigmaTokenManager__edit-mode'>
                        <input
                            type='text'
                            value={tokenValue}
                            onChange={handleInputChange}
                            placeholder={displayValue || 'figd_...'}
                            className='FigmaTokenManager__input'
                            disabled={props.disabled}
                            aria-label='Figma Personal Access Token'
                        />
                        <button
                            type='button'
                            onClick={handleCancelClick}
                            disabled={props.disabled}
                            className='FigmaTokenManager__button FigmaTokenManager__button--secondary'
                        >
                            Cancel
                        </button>
                        <p className='FigmaTokenManager__note'>
                            Enter new token and click "Save" button at the bottom of the page. Leave empty to keep current token.
                        </p>
                    </div>
                ) : (
                    <div className='FigmaTokenManager__display-mode'>
                        <span className='FigmaTokenManager__token-display'>
                            {displayValue || <em>Not configured</em>}
                        </span>
                        <button
                            type='button'
                            onClick={handleEditClick}
                            disabled={props.disabled}
                            className='FigmaTokenManager__button FigmaTokenManager__button--secondary'
                        >
                            {displayValue ? 'Change Token' : 'Set Token'}
                        </button>
                    </div>
                )}
            </div>
        </div>
    )
}

export default FigmaTokenManager

