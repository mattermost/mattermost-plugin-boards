// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useRef, useState} from 'react'
import {useIntl} from 'react-intl'

import {ContentBlock} from '../../blocks/contentBlock'
import {createTextBlock} from '../../blocks/textBlock'
import {createImageBlock} from '../../blocks/imageBlock'
import mutator from '../../mutator'
import TextIcon from '../../widgets/icons/text'
import {MarkdownEditor} from '../markdownEditor'
import {FigmaUtils} from '../../figmaUtils'
import octoClient from '../../octoClient'
import {sendFlashMessage} from '../flashMessages'
import {Block} from '../../blocks/block'
import {useCardDetailContext} from '../cardDetail/cardDetailContext'

import {contentRegistry} from './contentRegistry'

import './textElement.scss'

type Props = {
    block: ContentBlock
    readonly: boolean
}

const TextElement = (props: Props): JSX.Element => {
    const {block, readonly} = props
    const intl = useIntl()
    const containerRef = useRef<HTMLDivElement>(null)
    const cardDetail = useCardDetailContext()
    const [processingLinks, setProcessingLinks] = useState<Set<string>>(new Set())

    useEffect(() => {
        if (!readonly || !containerRef.current) {
            return
        }

        const container = containerRef.current
        const previewElement = container.querySelector('.octo-editor-preview')
        if (!previewElement) {
            return
        }

        // Find all links in the preview
        const links = previewElement.querySelectorAll('a')
        const figmaLinks: Array<{element: HTMLAnchorElement; url: string; hash: string}> = []

        links.forEach((link) => {
            const url = link.href
            const parsed = FigmaUtils.parseFigmaUrl(url)
            if (parsed) {
                const hash = FigmaUtils.generateHash(url)
                figmaLinks.push({element: link, url, hash})
            }
        })

        if (figmaLinks.length === 0) {
            return
        }

        // Get the card from context
        const card = cardDetail.card
        if (!card) {
            return
        }

        // Add [Attach preview] buttons for Figma links
        figmaLinks.forEach(({element, url, hash}) => {
            if (processingLinks.has(url)) {
                return
            }

            // Add [Attach preview] button after the link
            const existingButton = element.nextElementSibling
            if (existingButton?.classList.contains('figma-attach-button')) {
                return
            }

            const button = document.createElement('button')
            button.className = 'figma-attach-button'
            button.textContent = '[Attach preview]'
            button.onclick = async (e) => {
                e.preventDefault()
                e.stopPropagation()
                await handleAttachPreview(url, hash, button)
            }

            element.parentNode?.insertBefore(button, element.nextSibling)
        })
    }, [readonly, block.title, cardDetail, processingLinks])

    const handleAttachPreview = async (url: string, hash: string, button: HTMLButtonElement) => {
        const parsed = FigmaUtils.parseFigmaUrl(url)
        if (!parsed) {
            return
        }

        const card = cardDetail.card
        if (!card) {
            return
        }

        // Mark this link as being processed
        setProcessingLinks((prev) => new Set(prev).add(url))
        button.disabled = true
        button.textContent = '[Generating...]'

        try {
            // Call backend to generate preview
            const result = await octoClient.generateFigmaPreview(parsed.fileKey, parsed.nodeId, card.boardId)

            if (result.error) {
                sendFlashMessage({content: result.error, severity: 'high'})
                button.textContent = '[Attach preview]'
                button.disabled = false
                setProcessingLinks((prev) => {
                    const newSet = new Set(prev)
                    newSet.delete(url)
                    return newSet
                })
                return
            }

            // Create image block with the returned fileId
            const imageBlock = createImageBlock()
            imageBlock.parentId = card.id
            imageBlock.boardId = card.boardId
            imageBlock.fields.fileId = result.fileId
            imageBlock.title = hash

            // Find the index of the current text block
            const textBlockIndex = card.fields.contentOrder?.indexOf(block.id) ?? -1
            const insertIndex = textBlockIndex + 1

            const description = intl.formatMessage({id: 'TextElement.attachFigmaPreview', defaultMessage: 'attach Figma preview'})

            await mutator.performAsUndoGroup(async () => {
                const afterRedo = async (newBlock: Block) => {
                    const contentOrder = card.fields.contentOrder.slice()
                    contentOrder.splice(insertIndex, 0, newBlock.id)
                    await octoClient.patchBlock(card.boardId, card.id, {updatedFields: {contentOrder}})
                }

                const beforeUndo = async () => {
                    const contentOrder = card.fields.contentOrder.slice()
                    await octoClient.patchBlock(card.boardId, card.id, {updatedFields: {contentOrder}})
                }

                await mutator.insertBlock(imageBlock.boardId, imageBlock, description, afterRedo, beforeUndo)
            })

            // Remove the button
            button.remove()
            sendFlashMessage({content: intl.formatMessage({id: 'TextElement.figmaPreviewAttached', defaultMessage: 'Figma preview attached successfully'}), severity: 'low'})
        } catch (error) {
            sendFlashMessage({content: intl.formatMessage({id: 'TextElement.figmaPreviewFailed', defaultMessage: 'Failed to attach Figma preview'}), severity: 'high'})
            button.textContent = '[Attach preview]'
            button.disabled = false
            setProcessingLinks((prev) => {
                const newSet = new Set(prev)
                newSet.delete(url)
                return newSet
            })
        }
    }

    return (
        <div ref={containerRef}>
            <MarkdownEditor
                text={block.title}
                placeholderText={intl.formatMessage({id: 'ContentBlock.editText', defaultMessage: 'Edit text...'})}
                onBlur={(text) => {
                    if (text !== block.title) {
                        mutator.changeBlockTitle(block.boardId, block.id, block.title, text, intl.formatMessage({id: 'ContentBlock.editCardText', defaultMessage: 'edit card text'}))
                    }
                }}
                readonly={readonly}
            />
        </div>
    )
}

contentRegistry.registerContentType({
    type: 'text',
    getDisplayText: (intl) => intl.formatMessage({id: 'ContentBlock.text', defaultMessage: 'text'}),
    getIcon: () => <TextIcon/>,
    createBlock: async () => {
        return createTextBlock()
    },
    createComponent: (block, readonly) => {
        return (
            <TextElement
                block={block}
                readonly={readonly}
            />
        )
    },
})

export default React.memo(TextElement)
