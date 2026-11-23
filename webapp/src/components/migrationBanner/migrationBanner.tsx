// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react'
import {FormattedDate, FormattedMessage, useIntl} from 'react-intl'

import {useAppDispatch, useAppSelector} from '../../store/hooks'
import {fetchBoardMigration, getBoardMigration} from '../../store/boardMigration'
import {useLegacyMode} from '../../hooks/useProductName'
import {
    BoardsV2Summary,
    canMigrateFromSummary,
    dismissMigrationBanner,
    fetchBoardsV2Summary,
    isMigrationBannerDismissed,
    migrateDialogPath,
    navigateInMattermost,
    newBoardPath,
    siteURL,
} from '../../migration/boardsV2'
import Button from '../../widgets/buttons/button'

import './migrationBanner.scss'

type Props = {
    boardId: string
    isTemplate: boolean
}

const MigrationBanner = (props: Props): React.JSX.Element | null => {
    const {boardId, isTemplate} = props
    const intl = useIntl()
    const dispatch = useAppDispatch()
    const legacyMode = useLegacyMode()
    const migration = useAppSelector(getBoardMigration(boardId))
    const [summary, setSummary] = useState<BoardsV2Summary | null>(null)
    const [dismissed, setDismissed] = useState(() => isMigrationBannerDismissed(boardId))

    const notMoved = Boolean(migration && !migration.moved)

    useEffect(() => {
        if (legacyMode && boardId && !isTemplate) {
            dispatch(fetchBoardMigration(boardId))
        }
    }, [legacyMode, boardId, isTemplate])

    useEffect(() => {
        if (!legacyMode || !boardId || isTemplate) {
            return undefined
        }
        const refresh = () => dispatch(fetchBoardMigration(boardId))
        window.addEventListener('focus', refresh)
        return () => window.removeEventListener('focus', refresh)
    }, [legacyMode, boardId, isTemplate])

    useEffect(() => {
        setDismissed(isMigrationBannerDismissed(boardId))
    }, [boardId])

    useEffect(() => {
        if (!legacyMode || !notMoved || isTemplate) {
            return undefined
        }
        let cancelled = false
        fetchBoardsV2Summary().then((result) => {
            if (!cancelled) {
                setSummary(result)
            }
        })
        return () => {
            cancelled = true
        }
    }, [legacyMode, notMoved, isTemplate])

    const onDismiss = useCallback(() => {
        dismissMigrationBanner(boardId)
        setDismissed(true)
    }, [boardId])

    if (!legacyMode || !boardId || isTemplate || !migration) {
        return null
    }

    const label = intl.formatMessage({id: 'MigrationBanner.label', defaultMessage: 'Board migration'})

    if (migration.moved) {
        const canOpen = migration.can_open && Boolean(migration.team_name) && Boolean(migration.channel_name)
        const path = canOpen ? newBoardPath(migration.team_name!, migration.channel_name!) : ''
        const date = (
            <FormattedDate
                value={new Date(migration.moved_at || 0)}
                year='numeric'
                month='short'
                day='numeric'
            />
        )

        return (
            <section
                className='MigrationBanner'
                aria-label={label}
            >
                <p
                    className='MigrationBanner__text'
                    role='status'
                >
                    {migration.read_only ? (
                        <FormattedMessage
                            id='MigrationBanner.migratedReadOnly'
                            defaultMessage='This board was migrated to the new version of Boards on {date}. This board is now read-only here.'
                            values={{date}}
                        />
                    ) : (
                        <FormattedMessage
                            id='MigrationBanner.migrated'
                            defaultMessage="This board was migrated to the new version of Boards on {date}. Changes made here won't be copied over."
                            values={{date}}
                        />
                    )}
                    {!canOpen && (
                        <>
                            {' '}
                            <FormattedMessage
                                id='MigrationBanner.askToJoin'
                                defaultMessage='Ask a board member to add you to the new board.'
                            />
                        </>
                    )}
                </p>
                {canOpen && (
                    <div className='MigrationBanner__actions'>
                        <a
                            className='Button emphasis--primary size--small'
                            href={siteURL(path)}
                            onClick={(e) => navigateInMattermost(e, path)}
                        >
                            <span>
                                <FormattedMessage
                                    id='MigrationBanner.openNewBoard'
                                    defaultMessage='Open new board'
                                />
                            </span>
                        </a>
                    </div>
                )}
            </section>
        )
    }

    if (dismissed || !canMigrateFromSummary(summary)) {
        return null
    }

    const path = migrateDialogPath(boardId)

    return (
        <section
            className='MigrationBanner'
            aria-label={label}
        >
            <p
                className='MigrationBanner__text'
                role='status'
            >
                <FormattedMessage
                    id='MigrationBanner.notMigrated'
                    defaultMessage="There's a new version of Boards. Migrate this board to keep working on it with your team."
                />
            </p>
            <div className='MigrationBanner__actions'>
                <a
                    className='Button emphasis--primary size--small'
                    href={siteURL(path)}
                    onClick={(e) => navigateInMattermost(e, path)}
                >
                    <span>
                        <FormattedMessage
                            id='MigrationBanner.migrateBoard'
                            defaultMessage='Migrate this board'
                        />
                    </span>
                </a>
                <Button
                    emphasis='tertiary'
                    size='small'
                    onClick={onDismiss}
                >
                    <FormattedMessage
                        id='MigrationBanner.notNow'
                        defaultMessage='Not now'
                    />
                </Button>
            </div>
        </section>
    )
}

export default MigrationBanner
