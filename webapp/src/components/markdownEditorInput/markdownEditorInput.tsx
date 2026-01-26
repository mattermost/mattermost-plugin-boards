// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Editor from '@draft-js-plugins/editor'
import createEmojiPlugin from '@draft-js-plugins/emoji'
import '@draft-js-plugins/emoji/lib/plugin.css'
import createMentionPlugin from '@draft-js-plugins/mention'
import '@draft-js-plugins/mention/lib/plugin.css'
import {ContentState, DraftHandleValue, EditorState, getDefaultKeyBinding, Modifier, SelectionState} from 'draft-js'
import React, {
    ReactElement, useCallback, useEffect,
    useMemo, useRef,
    useState,
} from 'react'

import {debounce} from 'lodash'

import {useAppSelector} from '../../store/hooks'
import {IUser} from '../../user'
import {getBoardUsersList, getMe} from '../../store/users'
import createLiveMarkdownPlugin from '../live-markdown-plugin/liveMarkdownPlugin'
import {useHasPermissions} from '../../hooks/permissions'
import {Permission} from '../../constants'
import {BoardMember, BoardTypeOpen, MemberRole} from '../../blocks/board'
import mutator from '../../mutator'
import ConfirmAddUserForNotifications from '../confirmAddUserForNotifications'
import RootPortal from '../rootPortal'

import './markdownEditorInput.scss'

import {getCurrentBoard} from '../../store/boards'
import octoClient from '../../octoClient'

import {Utils} from '../../utils'
import {ClientConfig} from '../../config/clientConfig'
import {getClientConfig} from '../../store/clientConfig'

import Entry from './entryComponent/entryComponent'
import FormattingToolbar from './formattingToolbar'

const imageURLForUser = (window as any).Components?.imageURLForUser

type MentionUser = {
    user: IUser
    name: string
    avatar: string
    is_bot: boolean
    is_guest: boolean
    displayName: string
    isBoardMember: boolean
}

type Props = {
    onChange?: (text: string) => void
    onFocus?: () => void
    onBlur?: (text: string) => void
    onEditorCancel?: () => void
    initialText?: string
    id?: string
    isEditing: boolean
    saveOnEnter?: boolean
    showToolbar?: boolean
}

const MarkdownEditorInput = (props: Props): ReactElement => {
    const {onChange, onFocus, onBlur, initialText, id} = props
    const boardUsers = useAppSelector<IUser[]>(getBoardUsersList)
    const board = useAppSelector(getCurrentBoard)
    const clientConfig = useAppSelector<ClientConfig>(getClientConfig)
    const ref = useRef<Editor>(null)
    const allowManageBoardRoles = useHasPermissions(board.teamId, board.id, [Permission.ManageBoardRoles])
    const [confirmAddUser, setConfirmAddUser] = useState<IUser|null>(null)
    const me = useAppSelector<IUser|null>(getMe)

    const [suggestions, setSuggestions] = useState<MentionUser[]>([])

    const loadSuggestions = async (term: string) => {
        let users: IUser[]

        if (!me?.is_guest && (allowManageBoardRoles || (board && board.type === BoardTypeOpen))) {
            const excludeBots = true
            users = await octoClient.searchTeamUsers(term, excludeBots)
        } else {
            users = boardUsers.
                filter((user) => {
                    // no search term
                    if (!term) {
                        return true
                    }

                    // does the search term occur anywhere in the display name?
                    return Utils.getUserDisplayName(user, clientConfig.teammateNameDisplay).includes(term)
                }).

                // first 10 results
                slice(0, 10)
        }

        const mentions: MentionUser[] = users.map(
            (user: IUser): MentionUser => ({
                name: user.username,
                avatar: `${imageURLForUser ? imageURLForUser(user.id) : ''}`,
                is_bot: user.is_bot,
                is_guest: user.is_guest,
                displayName: Utils.getUserDisplayName(user, clientConfig.teammateNameDisplay),
                isBoardMember: Boolean(boardUsers.find((u) => u.id === user.id)),
                user,
            }))
        setSuggestions(mentions)
    }

    const debouncedLoadSuggestion = useMemo(() => debounce(loadSuggestions, 200), [])

    useEffect(() => {
        // Get the ball rolling. Searching for empty string
        // returns first 10 users in alphabetical order.
        loadSuggestions('')
    }, [])

    const generateEditorState = (text?: string) => {
        const state = EditorState.createWithContent(ContentState.createFromText(text || ''))
        return EditorState.moveSelectionToEnd(state)
    }

    const [editorState, setEditorState] = useState(() => generateEditorState(initialText))

    const addUser = useCallback(async (userId: string, role: string) => {
        const newRole = role || MemberRole.Viewer
        const newMember = {
            boardId: board.id,
            userId,
            roles: role,
            schemeAdmin: newRole === MemberRole.Admin,
            schemeEditor: newRole === MemberRole.Admin || newRole === MemberRole.Editor,
            schemeCommenter: newRole === MemberRole.Admin || newRole === MemberRole.Editor || newRole === MemberRole.Commenter,
            schemeViewer: newRole === MemberRole.Admin || newRole === MemberRole.Editor || newRole === MemberRole.Commenter || newRole === MemberRole.Viewer,
        } as BoardMember

        setConfirmAddUser(null)
        setEditorState(EditorState.moveSelectionToEnd(editorState))
        ref.current?.focus()
        await mutator.createBoardMember(newMember)
    }, [board, editorState])

    const [initialTextCache, setInitialTextCache] = useState<string | undefined>(initialText)
    const [initialTextUsed, setInitialTextUsed] = useState<boolean>(false)

    // avoiding stale closure
    useEffect(() => {
        // only change editor state when initialText actually changes from one defined value to another.
        // This is needed to make the mentions plugin work. For some reason, if we don't check
        // for this if condition here, mentions don't work. I suspect it's because without
        // the in condition, we're changing editor state twice during component initialization
        // and for some reason it causes mentions to not show up.

        // initial text should only be used once, i'e', initially
        // `initialTextUsed` flag records if the initialText prop has been used
        // to se the editor state once as a truthy value.
        // Once used, we don't react to its changing value

        if (!initialTextUsed && initialText && initialText !== initialTextCache) {
            setEditorState(generateEditorState(initialText || ''))
            setInitialTextCache(initialText)
            setInitialTextUsed(true)
        }
    }, [initialText])

    const [isMentionPopoverOpen, setIsMentionPopoverOpen] = useState(false)
    const [isEmojiPopoverOpen, setIsEmojiPopoverOpen] = useState(false)

    const {MentionSuggestions, plugins, EmojiSuggestions} = useMemo(() => {
        const mentionPlugin = createMentionPlugin({mentionPrefix: '@'})
        const emojiPlugin = createEmojiPlugin()
        const markdownPlugin = createLiveMarkdownPlugin()

        // eslint-disable-next-line @typescript-eslint/no-shadow
        const {EmojiSuggestions} = emojiPlugin
        // eslint-disable-next-line @typescript-eslint/no-shadow
        const {MentionSuggestions} = mentionPlugin
        // eslint-disable-next-line @typescript-eslint/no-shadow
        const plugins = [
            mentionPlugin,
            emojiPlugin,
            markdownPlugin,
        ]
        return {plugins, MentionSuggestions, EmojiSuggestions}
    }, [])

    const onEditorStateChange = useCallback((newEditorState: EditorState) => {
        // newEditorState.
        const newText = newEditorState.getCurrentContent().getPlainText()

        onChange && onChange(newText)
        setEditorState(newEditorState)
    }, [onChange])

    const customKeyBindingFn = useCallback((e: React.KeyboardEvent) => {
        if (isMentionPopoverOpen || isEmojiPopoverOpen) {
            return undefined
        }

        if (e.key === 'Escape') {
            return 'editor-blur'
        }

        if (e.key === 'Backspace') {
            return 'backspace'
        }

        // Formatting shortcuts (Ctrl+B, Ctrl+I, etc.)
        if (e.ctrlKey || e.metaKey) {
            switch (e.key.toLowerCase()) {
            case 'b':
                return 'format-bold'
            case 'i':
                return 'format-italic'
            case 'u':
                return 'format-strikethrough'
            case 'k':
                return 'format-link'
            case 'e':
                return 'format-code'
            }
        }

        if (getDefaultKeyBinding(e) === 'undo') {
            return 'editor-undo'
        }

        if (getDefaultKeyBinding(e) === 'redo') {
            return 'editor-redo'
        }

        return getDefaultKeyBinding(e as any)
    }, [isEmojiPopoverOpen, isMentionPopoverOpen])

    const onEditorStateBlur = useCallback(() => {
        if (confirmAddUser) {
            return
        }
        const text = editorState.getCurrentContent().getPlainText()
        onBlur && onBlur(text)
    }, [editorState.getCurrentContent().getPlainText(), onBlur, confirmAddUser])

    const onMentionPopoverOpenChange = useCallback((open: boolean) => {
        setIsMentionPopoverOpen(open)
    }, [])

    const onEmojiPopoverOpen = useCallback(() => {
        setIsEmojiPopoverOpen(true)
    }, [])

    const onEmojiPopoverClose = useCallback(() => {
        setIsEmojiPopoverOpen(false)
    }, [])

    const onSearchChange = useCallback(({value}: { value: string }) => {
        debouncedLoadSuggestion(value)
    }, [suggestions])

    const handleFormat = useCallback((format: string, stateToUse?: EditorState) => {
        const currentEditorState = stateToUse || editorState
        const selection = currentEditorState.getSelection()
        const currentContent = currentEditorState.getCurrentContent()
        let newState = currentEditorState

        // Get selected text for placeholder text
        const startKey = selection.getStartKey()
        const endKey = selection.getEndKey()
        const startOffset = selection.getStartOffset()
        const endOffset = selection.getEndOffset()

        // Check if selection is collapsed (no text selected)
        const isCollapsed = selection.isCollapsed()

        // Get selected text (works for single and multi-line selections)
        let selectedText = ''
        if (!isCollapsed) {
            if (startKey === endKey) {
                // Single line selection
                const startBlock = currentContent.getBlockForKey(startKey)
                selectedText = startBlock.getText().slice(startOffset, endOffset)
            } else {
                // Multi-line selection - collect text from all blocks
                const blockMap = currentContent.getBlockMap()
                let foundStart = false
                const textParts: string[] = []

                blockMap.forEach((block) => {
                    if (!block) {
                        return
                    }

                    const blockKey = block.getKey()

                    if (blockKey === startKey) {
                        foundStart = true
                    }

                    if (foundStart) {
                        if (blockKey === startKey) {
                            // First block - take from startOffset to end
                            textParts.push(block.getText().slice(startOffset))
                        } else if (blockKey === endKey) {
                            // Last block - take from start to endOffset
                            textParts.push(block.getText().slice(0, endOffset))
                        } else {
                            // Middle blocks - take all text
                            textParts.push(block.getText())
                        }
                    }

                    if (blockKey === endKey) {
                        foundStart = false
                    }
                })

                selectedText = textParts.join('\n')
            }
        }

        switch (format) {
        case 'bold':
        case 'italic':
        case 'strikethrough':
        case 'code': {
            // For inline styles, wrap with markdown syntax
            const markers = {
                bold: '**',
                italic: '*',
                strikethrough: '~~',
                code: '`',
            }
            const marker = markers[format as keyof typeof markers]
            const placeholder = {
                bold: 'bold text',
                italic: 'italic text',
                strikethrough: 'strikethrough text',
                code: 'code',
            }
            const text = selectedText || placeholder[format as keyof typeof placeholder]
            const formattedText = `${marker}${text}${marker}`

            const contentWithText = Modifier.replaceText(
                currentContent,
                selection,
                formattedText,
            )
            newState = EditorState.push(editorState, contentWithText, 'insert-characters')
            break
        }
        case 'link': {
            const linkText = selectedText || 'link text'
            const formattedText = `[${linkText}](url)`
            const contentWithText = Modifier.replaceText(
                currentContent,
                selection,
                formattedText,
            )
            newState = EditorState.push(editorState, contentWithText, 'insert-characters')
            break
        }
        case 'bulletList':
        case 'numberList': {
            // For lists, add prefix to all selected lines
            const prefix = format === 'bulletList' ? '* ' : '1. '

            // Get all blocks in selection
            const blockMap = currentContent.getBlockMap()
            const startBlockKey = selection.getStartKey()
            const endBlockKey = selection.getEndKey()

            let newContent = currentContent
            let foundStart = false
            let listNumber = 1

            blockMap.forEach((block) => {
                if (!block) {
                    return
                }

                const blockKey = block.getKey()

                // Start processing from startBlockKey
                if (blockKey === startBlockKey) {
                    foundStart = true
                }

                // Process blocks in selection
                if (foundStart) {
                    const blockText = block.getText()
                    const actualPrefix = format === 'bulletList' ? prefix : `${listNumber}. `

                    // Check if text already has the same type of prefix
                    const bulletRegex = /^\* /
                    const numberRegex = /^\d+\. /

                    let newText: string
                    if (format === 'bulletList') {
                        // If already has bullet prefix, skip. If has number prefix, replace it.
                        if (bulletRegex.test(blockText)) {
                            newText = blockText
                        } else if (numberRegex.test(blockText)) {
                            newText = blockText.replace(numberRegex, prefix)
                        } else {
                            newText = prefix + blockText
                        }
                    } else {
                        // numberList: If already has number prefix, skip. If has bullet prefix, replace it.
                        if (numberRegex.test(blockText)) {
                            newText = blockText
                        } else if (bulletRegex.test(blockText)) {
                            newText = blockText.replace(bulletRegex, actualPrefix)
                        } else {
                            newText = actualPrefix + blockText
                        }
                    }

                    // Create a forward selection for this block to avoid backward selection issues
                    const blockSelection = SelectionState.createEmpty(blockKey).merge({
                        anchorOffset: 0,
                        focusOffset: blockText.length,
                    }) as SelectionState

                    newContent = Modifier.replaceText(
                        newContent,
                        blockSelection,
                        newText,
                    )

                    listNumber++
                }

                // Stop after endBlockKey
                if (blockKey === endBlockKey) {
                    foundStart = false
                }
            })

            newState = EditorState.push(editorState, newContent, 'insert-characters')
            break
        }
        case 'quote': {
            // For quote, add prefix to all selected lines
            const prefix = '> '

            // Get all blocks in selection
            const blockMap = currentContent.getBlockMap()
            const startBlockKey = selection.getStartKey()
            const endBlockKey = selection.getEndKey()

            let newContent = currentContent
            let foundStart = false

            blockMap.forEach((block) => {
                if (!block) {
                    return
                }

                const blockKey = block.getKey()

                // Start processing from startBlockKey
                if (blockKey === startBlockKey) {
                    foundStart = true
                }

                // Process blocks in selection
                if (foundStart) {
                    const blockText = block.getText()

                    // Check if text already has a quote prefix
                    const quoteRegex = /^> /
                    const hasPrefix = quoteRegex.test(blockText)

                    // Only add prefix if it doesn't already exist
                    const newText = hasPrefix ? blockText : prefix + blockText

                    // Create a forward selection for this block to avoid backward selection issues
                    const blockSelection = SelectionState.createEmpty(blockKey).merge({
                        anchorOffset: 0,
                        focusOffset: blockText.length,
                    }) as SelectionState

                    newContent = Modifier.replaceText(
                        newContent,
                        blockSelection,
                        newText,
                    )
                }

                // Stop after endBlockKey
                if (blockKey === endBlockKey) {
                    foundStart = false
                }
            })

            newState = EditorState.push(editorState, newContent, 'insert-characters')
            break
        }
        default:
            return
        }

        onEditorStateChange(newState)
        ref.current?.focus()
    }, [editorState, onEditorStateChange])

    const handleKeyCommand = useCallback((command: string, currentState: EditorState): DraftHandleValue => {
        if (command === 'editor-blur') {
            ref.current?.blur()
            return 'handled'
        }

        if (command === 'editor-redo') {
            const selectionRemovedState = EditorState.redo(currentState)
            onEditorStateChange(EditorState.redo(selectionRemovedState))

            return 'handled'
        }

        if (command === 'editor-undo') {
            const selectionRemovedState = EditorState.undo(currentState)
            onEditorStateChange(EditorState.undo(selectionRemovedState))

            return 'handled'
        }

        // Handle formatting shortcuts
        if (command === 'format-bold') {
            handleFormat('bold', currentState)
            return 'handled'
        }

        if (command === 'format-italic') {
            handleFormat('italic', currentState)
            return 'handled'
        }

        if (command === 'format-strikethrough') {
            handleFormat('strikethrough', currentState)
            return 'handled'
        }

        if (command === 'format-link') {
            handleFormat('link', currentState)
            return 'handled'
        }

        if (command === 'format-code') {
            handleFormat('code', currentState)
            return 'handled'
        }

        if (command === 'backspace') {
            if (props.onEditorCancel && editorState.getCurrentContent().getPlainText().length === 0) {
                props.onEditorCancel()
                return 'handled'
            }
        }

        return 'not-handled'
    }, [props.onEditorCancel, editorState, handleFormat])

    const className = 'MarkdownEditorInput'

    const handleReturn = (e: any, state: EditorState): DraftHandleValue => {
        if (!e.shiftKey) {
            const text = state.getCurrentContent().getPlainText()
            onBlur && onBlur(text)
            return 'handled'
        }
        return 'not-handled'
    }

    return (
        <div
            className={className}
            onKeyDown={(e: React.KeyboardEvent) => {
                if (isMentionPopoverOpen || isEmojiPopoverOpen) {
                    e.stopPropagation()
                }
            }}
        >
            <Editor
                editorKey={id}
                editorState={editorState}
                onChange={onEditorStateChange}
                plugins={plugins}
                ref={ref}
                onBlur={onEditorStateBlur}
                onFocus={onFocus}
                keyBindingFn={customKeyBindingFn}
                handleKeyCommand={handleKeyCommand}
                handleReturn={props.saveOnEnter ? handleReturn : undefined}
            />
            <MentionSuggestions
                open={isMentionPopoverOpen}
                onOpenChange={onMentionPopoverOpenChange}
                suggestions={suggestions}
                onSearchChange={onSearchChange}
                entryComponent={Entry}
                onAddMention={(mention) => {
                    if (mention.isBoardMember) {
                        return
                    }
                    setConfirmAddUser(mention.user)
                }}
            />
            {props.showToolbar && <FormattingToolbar onFormat={handleFormat}/>}
            <EmojiSuggestions
                onOpen={onEmojiPopoverOpen}
                onClose={onEmojiPopoverClose}
            />
            {confirmAddUser &&
                <RootPortal>
                    <ConfirmAddUserForNotifications
                        allowManageBoardRoles={allowManageBoardRoles}
                        minimumRole={board.minimumRole}
                        user={confirmAddUser}
                        onConfirm={addUser}
                        onClose={() => {
                            setConfirmAddUser(null)
                            setEditorState(EditorState.moveSelectionToEnd(editorState))
                            ref.current?.focus()
                        }}
                    />
                </RootPortal>}
        </div>
    )
}

export default MarkdownEditorInput
