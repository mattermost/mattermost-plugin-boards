// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'

import {PropertyProps} from '../types'
import BaseTextEditor from '../baseTextEditor'

const Phone = (props: PropertyProps): JSX.Element => {
    return (
        <BaseTextEditor
            {...props}
            validator={() => true}
        />
    )
}
export default Phone
