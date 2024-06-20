// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
import React from 'react'

import {useIntl, FormattedMessage} from 'react-intl'

import IconButton from '../../widgets/buttons/iconButton'
import Button from '../../widgets/buttons/button'

import CloseIcon from '../../widgets/icons/close'

import {useAppSelector, useAppDispatch} from '../../store/hooks'
import octoClient from '../../octoClient'
import {IUser, UserConfigPatch} from '../../user'
import {getMe, patchProps, getVersionMessageCanceled, versionProperty} from '../../store/users'

import CompassIcon from '../../widgets/icons/compassIcon'
import TelemetryClient, {TelemetryCategory, TelemetryActions} from '../../telemetry/telemetryClient'

import './versionMessage.scss'
const helpURL = 'https://mattermost.com/pl/whats-new-boards/'

const VersionMessage = React.memo(() => {
    const intl = useIntl()
    const dispatch = useAppDispatch()
    const me = useAppSelector<IUser|null>(getMe)
    const versionMessageCanceled = useAppSelector(getVersionMessageCanceled)

    if (!me || me.id === 'single-user' || versionMessageCanceled) {
        return null
    }

    const closeDialogText = intl.formatMessage({
        id: 'Dialog.closeDialog',
        defaultMessage: 'Close dialog',
    })

    const onClose = async () => {
        if (me) {
            const patch: UserConfigPatch = {
                updatedFields: {
                    [versionProperty]: 'true',
                },
            }
            const patchedProps = await octoClient.patchUserConfig(me.id, patch)
            if (patchedProps) {
                dispatch(patchProps(patchedProps))
            }
        }
    }

    return (
        <div className='VersionMessage'>
            <div className='banner'>
                <CompassIcon
                    icon='information-outline'
                    className='CompassIcon'
                />
                <FormattedMessage
                    id='VersionMessage.help'
                    defaultMessage="Check out what's new in this version."
                />

                <Button
                    title='Learn more'
                    size='xsmall'
                    emphasis='primary'
                    onClick={() => {
                        TelemetryClient.trackEvent(TelemetryCategory, TelemetryActions.VersionMoreInfo)
                        window.open(helpURL)
                    }}
                >
                    <FormattedMessage
                        id='VersionMessage.learn-more'
                        defaultMessage='Learn more'
                    />
                </Button>

            </div>

            <IconButton
                className='margin-right'
                onClick={onClose}
                icon={<CloseIcon/>}
                title={closeDialogText}
                size='small'
            />
        </div>
    )
})

VersionMessage.displayName = 'VersionMessage'

export default VersionMessage
