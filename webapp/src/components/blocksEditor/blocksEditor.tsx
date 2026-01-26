// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useMemo, useCallback, useEffect} from 'react'
import {DndProvider} from 'react-dnd'
import {HTML5Backend} from 'react-dnd-html5-backend'

import Editor from './editor'
import {BlockData} from './blocks/types'
import BlockContent from './blockContent'
import * as registry from './blocks'

type Props = {
    boardId?: string
    onBlockCreated: (block: BlockData, afterBlock?: BlockData) => Promise<BlockData|null>
    onBlockModified: (block: BlockData) => Promise<BlockData|null>
    onBlockMoved: (block: BlockData, beforeBlock: BlockData|null, afterBlock: BlockData|null) => Promise<void>
    blocks: BlockData[]
    onEditingChange?: (blockId: string | null) => void
    focusBlockId?: string | null
}

function BlocksEditor(props: Props) {
    const [nextType, setNextType] = useState<string>('')
    const [editing, setEditing] = useState<BlockData|null>(null)
    const [afterBlock, setAfterBlock] = useState<BlockData|null>(null)
    const contentOrder = useMemo(() => props.blocks.filter((b) => b.id).map((b) => b.id!), [props.blocks])

    const setEditingWithCallback = useCallback((block: BlockData | null) => {
        setEditing(block)
        if (props.onEditingChange) {
            props.onEditingChange(block?.id || null)
        }
    }, [props.onEditingChange])

    useEffect(() => {
        if (props.focusBlockId) {
            const blockToFocus = props.blocks.find((b) => b.id === props.focusBlockId)
            if (blockToFocus) {
                setEditingWithCallback(blockToFocus)
                setAfterBlock(null)
            }
        }
    }, [props.focusBlockId, props.blocks, setEditingWithCallback])

    return (
        <div
            className='BlocksEditor'
            onKeyDown={(e: React.KeyboardEvent<HTMLDivElement>) => {
                if (e.key === 'ArrowUp') {
                    if (editing === null) {
                        if (afterBlock === null) {
                            setEditingWithCallback(props.blocks[props.blocks.length - 1] || null)
                        } else {
                            setEditingWithCallback(afterBlock)
                        }
                        setAfterBlock(null)
                        return
                    }
                    let prevBlock = null
                    for (const b of props.blocks) {
                        if (editing?.id === b.id) {
                            break
                        }
                        const blockType = registry.get(b.contentType)
                        if (blockType.editable) {
                            prevBlock = b
                        }
                    }
                    if (prevBlock) {
                        setEditingWithCallback(prevBlock)
                        setAfterBlock(null)
                    }
                } else if (e.key === 'ArrowDown') {
                    let currentBlock = editing
                    if (currentBlock === null) {
                        currentBlock = afterBlock
                    }
                    if (currentBlock === null) {
                        return
                    }

                    let nextBlock = null
                    let breakNext = false
                    for (const b of props.blocks) {
                        if (breakNext) {
                            const blockType = registry.get(b.contentType)
                            if (blockType.editable) {
                                nextBlock = b
                                break
                            }
                        }
                        if (currentBlock.id === b.id) {
                            breakNext = true
                        }
                    }
                    setEditingWithCallback(nextBlock)
                    setAfterBlock(null)
                }
            }}
        >
            <DndProvider backend={HTML5Backend}>
                {Object.values(props.blocks).map((d) => (
                    <div
                        key={d.id}
                    >
                        <BlockContent
                            key={d.id}
                            block={d}
                            editing={editing}
                            setEditing={(block) => {
                                setEditingWithCallback(block)
                                setAfterBlock(null)
                            }}
                            contentOrder={contentOrder}
                            setAfterBlock={setAfterBlock}
                            onSave={async (b) => {
                                const newBlock = await props.onBlockModified(b)
                                setNextType(registry.get(b.contentType).nextType || '')
                                setAfterBlock(newBlock)
                                return newBlock
                            }}
                            onMove={props.onBlockMoved}
                        />
                        {afterBlock && afterBlock.id === d.id && (
                            <Editor
                                initialValue=''
                                initialContentType={nextType}
                                onSave={async (b) => {
                                    const newBlock = await props.onBlockCreated(b, afterBlock)
                                    setNextType(registry.get(b.contentType).nextType || '')
                                    setAfterBlock(newBlock)
                                    return newBlock
                                }}
                            />)}
                    </div>
                ))}
                {!editing && !afterBlock && <Editor onSave={props.onBlockCreated}/>}
            </DndProvider>
        </div>
    )
}

export default BlocksEditor
