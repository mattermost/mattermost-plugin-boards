// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import {useEffect, useCallback} from 'react'
import {useIntl} from 'react-intl'

import {ImageBlock, createImageBlock} from '../../blocks/imageBlock'
import {createTextBlock} from '../../blocks/textBlock'
import {sendFlashMessage} from '../flashMessages'
import {Block} from '../../blocks/block'
import octoClient from '../../octoClient'
import mutator from '../../mutator'

type EditingContext = {
    blockId: string | null
    blockIndex: number
}

type ImagePasteOptions = {
    getEditingContext?: () => EditingContext
    onImageInserted?: (newTextBlockId: string) => void
}

export default function useImagePaste(
    boardId: string,
    cardId: string,
    contentOrder: Array<string | string[]>,
    options?: ImagePasteOptions
): void {
    const intl = useIntl()
    const uploadItems = useCallback(async (items: FileList) => {
        let newImage: File|null = null
        const uploads: Array<Promise<string|undefined>> = []

        if (!items.length) {
            return
        }

        for (const item of items) {
            newImage = item
            if (newImage?.type.indexOf('image/') === 0) {
                uploads.push(octoClient.uploadFile(boardId, newImage))
            }
        }

        const uploaded = await Promise.all(uploads)
        const blocksToInsert: Block[] = []
        let someFilesNotUploaded = false

        const editingContext = options?.getEditingContext?.() || {blockId: null, blockIndex: -1}
        const insertIndex = editingContext.blockIndex >= 0 ? editingContext.blockIndex + 1 : contentOrder.length

        for (const fileId of uploaded) {
            if (!fileId) {
                someFilesNotUploaded = true
                continue
            }
            const imageBlock = createImageBlock()
            imageBlock.parentId = cardId
            imageBlock.boardId = boardId
            imageBlock.fields.fileId = fileId || ''
            blocksToInsert.push(imageBlock)

            const textBlock = createTextBlock()
            textBlock.parentId = cardId
            textBlock.boardId = boardId
            blocksToInsert.push(textBlock)
        }

        if (someFilesNotUploaded) {
            sendFlashMessage({content: intl.formatMessage({id: 'imagePaste.upload-failed', defaultMessage: 'Some files not uploaded. File size limit reached'}), severity: 'normal'})
        }

        if (blocksToInsert.length === 0) {
            return
        }

        const afterRedo = async (newBlocks: Block[]) => {
            const newContentOrder = JSON.parse(JSON.stringify(contentOrder))
            newContentOrder.splice(insertIndex, 0, ...newBlocks.map((b: Block) => b.id))
            await octoClient.patchBlock(boardId, cardId, {updatedFields: {contentOrder: newContentOrder}})

            const lastTextBlock = newBlocks[newBlocks.length - 1]
            if (lastTextBlock && options?.onImageInserted) {
                options.onImageInserted(lastTextBlock.id)
            }
        }

        const beforeUndo = async () => {
            const newContentOrder = JSON.parse(JSON.stringify(contentOrder))
            await octoClient.patchBlock(boardId, cardId, {updatedFields: {contentOrder: newContentOrder}})
        }

        await mutator.insertBlocks(boardId, blocksToInsert, 'pasted images', afterRedo, beforeUndo)
    }, [cardId, contentOrder, boardId, options])

    const onDrop = useCallback((event: DragEvent): void => {
        if (event.dataTransfer) {
            const items = event.dataTransfer.files
            uploadItems(items)
        }
    }, [uploadItems])

    const onPaste = useCallback((event: ClipboardEvent): void => {
        if (event.clipboardData) {
            const items = event.clipboardData.files
            uploadItems(items)
        }
    }, [uploadItems])

    useEffect(() => {
        document.addEventListener('paste', onPaste)
        document.addEventListener('drop', onDrop)
        return () => {
            document.removeEventListener('paste', onPaste)
            document.removeEventListener('drop', onDrop)
        }
    }, [uploadItems, onPaste, onDrop])
}
