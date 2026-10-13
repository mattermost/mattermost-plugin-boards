// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import {IntlShape} from 'react-intl'

import {IAppWindow} from './types'
import {Block} from './blocks/block'
import {Board} from './blocks/board'
import mutator from './mutator'
import {Utils} from './utils'
import {sendFlashMessage} from './components/flashMessages'

declare let window: IAppWindow

class Archiver {
    static async exportBoardArchive(board: Board): Promise<void> {
        this.exportArchive(mutator.exportBoardArchive(board.id))
    }

    static async exportFullArchive(teamID: string): Promise<void> {
        this.exportArchive(mutator.exportFullArchive(teamID))
    }

    private static exportArchive(prom: Promise<Response>): void {
        // TODO:  don't download whole archive before presenting SaveAs dialog.
        prom.then((response) => {
            response.blob().
                then((blob) => {
                    const link = document.createElement('a')
                    link.style.display = 'none'

                    const date = new Date()
                    const filename = `archive-${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}.boardarchive`

                    const file = new Blob([blob], {type: 'application/octet-stream'})
                    link.href = URL.createObjectURL(file)
                    link.download = filename
                    document.body.appendChild(link)		// FireFox support

                    link.click()

                    // TODO: Review if this is needed in the future, this is to fix the problem with linux webview links
                    if (window.openInNewBrowser) {
                        window.openInNewBrowser(link.href)
                    }

                    // TODO: Remove or reuse link and revolkObjectURL to avoid memory leak
                })
        })
    }

    private static async importArchiveFromFile(file: File, intl: IntlShape): Promise<void> {
        sendFlashMessage({
            content: intl.formatMessage({id: 'ImportArchive.importing', defaultMessage: 'Importing archive…'}),
            severity: 'normal',
        })

        let response: Response
        try {
            response = await mutator.importFullArchive(file)
        } catch (e) {
            Utils.logError(`ERROR importing archive: ${e}`)
            sendFlashMessage({
                content: intl.formatMessage({id: 'ImportArchive.failed', defaultMessage: 'Import failed'}),
                severity: 'high',
            })
            return
        }

        if (response.status !== 200) {
            const errorMessage = await Archiver.readImportError(response)
            Utils.logError(`ERROR importing archive: ${response.status} ${errorMessage}`)
            sendFlashMessage({
                content: errorMessage ? intl.formatMessage({id: 'ImportArchive.failedWithReason', defaultMessage: 'Import failed: {error}'}, {error: errorMessage}) : intl.formatMessage({id: 'ImportArchive.failed', defaultMessage: 'Import failed'}),
                severity: 'high',
            })
            return
        }

        sendFlashMessage({
            content: intl.formatMessage({id: 'ImportArchive.succeeded', defaultMessage: 'Archive imported'}),
            severity: 'normal',
        })
    }

    private static async readImportError(response: Response): Promise<string> {
        try {
            const text = await response.text()
            if (!text) {
                return ''
            }

            try {
                const json = JSON.parse(text)
                return json?.error || text
            } catch {
                return text
            }
        } catch {
            return ''
        }
    }

    static isValidBlock(block: Block): boolean {
        if (!block.id || !block.boardId) {
            return false
        }

        return true
    }

    static importFullArchive(intl: IntlShape, onComplete?: () => void): void {
        const input = document.createElement('input')
        input.type = 'file'
        input.accept = '.boardarchive'
        input.onchange = async () => {
            const file = input.files && input.files[0]
            if (file) {
                await Archiver.importArchiveFromFile(file, intl)
            }

            onComplete?.()
        }

        input.style.display = 'none'
        document.body.appendChild(input)
        input.click()

        // TODO: Remove or reuse input
    }
}

export {Archiver}
