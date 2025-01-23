// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {IntlShape} from 'react-intl'

import {PropertyType, PropertyTypeEnum, FilterValueType} from '../types'

import UpdatedBy from './updatedBy'

export default class UpdatedByProperty extends PropertyType {
    Editor = UpdatedBy
    name = 'Last Modified By'
    type = 'updatedBy' as PropertyTypeEnum
    isReadOnly = true
    displayName = (intl: IntlShape) => intl.formatMessage({id: 'PropertyType.UpdatedBy', defaultMessage: 'Last updated by'})
    canFilter = true
    filterValueType = 'person' as FilterValueType
    canGroup = true
}
