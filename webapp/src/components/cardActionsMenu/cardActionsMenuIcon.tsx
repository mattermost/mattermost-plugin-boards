// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import OptionsIcon from '../../widgets/icons/options'
import IconButton from '../../widgets/buttons/iconButton'

import './cardActionsMenuIcon.scss'

const CardActionsMenuIcon = () => {
    return (
        <IconButton
            className='CardActionsMenuIcon'
            icon={<OptionsIcon/>}
        />
    )
}

export default CardActionsMenuIcon
