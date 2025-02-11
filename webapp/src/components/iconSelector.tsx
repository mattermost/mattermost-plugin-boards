// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'
import {useIntl} from 'react-intl'

import RandomIcon from '../widgets/icons/random'
import EmojiPicker from '../widgets/emojiPicker'
import DeleteIcon from '../widgets/icons/delete'
import EmojiIcon from '../widgets/icons/emoji'
import Menu from '../widgets/menu'
import MenuWrapper from '../widgets/menuWrapper'
import './iconSelector.scss'

type Props = {
    readonly?: boolean
    iconElement: JSX.Element 
    onAddRandomIcon: () => Promise<void>
    onSelectEmoji: (emoji: string) => void
    onRemoveIcon: () => Promise<void>
}

const IconSelector = React.memo((props: Props) => {
    const intl = useIntl()

    return (
        <div className='IconSelector'>
            {props.readonly && props.iconElement}
            {!props.readonly &&
                <MenuWrapper>
                    {props.iconElement}
                    <Menu>
                        <Menu.Text
                            id='random'
                            icon={<RandomIcon/>}
                            name={intl.formatMessage({id: 'ViewTitle.random-icon', defaultMessage: 'Random'})}
                            onClick={props.onAddRandomIcon}
                        />
                        <Menu.SubMenu
                            id='pick'
                            icon={<EmojiIcon/>}
                            name={intl.formatMessage({id: 'ViewTitle.pick-icon', defaultMessage: 'Pick icon'})}
                        >
                            <EmojiPicker onSelect={props.onSelectEmoji}/>
                        </Menu.SubMenu>
                        <Menu.Text
                            id='remove'
                            icon={<DeleteIcon/>}
                            name={intl.formatMessage({id: 'ViewTitle.remove-icon', defaultMessage: 'Remove icon'})}
                            onClick={props.onRemoveIcon}
                        />
                    </Menu>
                </MenuWrapper>
            }
        </div>
    )
})

IconSelector.displayName = 'IconSelector'

export default IconSelector
