// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {IntlShape} from 'react-intl'

import {PropertyType, PropertyTypeEnum, FilterValueType} from '../types'

import Email from './email'

export default class EmailProperty extends PropertyType {
    Editor = Email
    name = 'Email'
    type = 'email' as PropertyTypeEnum
    displayName = (intl: IntlShape) => intl.formatMessage({id: 'PropertyType.Email', defaultMessage: 'Email'})
    canFilter = true
    filterValueType = 'text' as FilterValueType
}
