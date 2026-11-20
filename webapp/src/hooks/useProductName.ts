// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {IntlShape, useIntl} from 'react-intl'

import {useAppSelector} from '../store/hooks'
import {getLegacyMode} from '../store/clientConfig'

export function getProductName(intl: IntlShape, legacyMode: boolean): string {
    if (legacyMode) {
        return intl.formatMessage({id: 'Product.name.legacy', defaultMessage: 'Legacy Boards'})
    }
    return intl.formatMessage({id: 'Product.name', defaultMessage: 'Boards'})
}

export const useLegacyMode = (): boolean => {
    return useAppSelector(getLegacyMode)
}

export const useProductName = (): string => {
    const intl = useIntl()
    const legacyMode = useLegacyMode()
    return getProductName(intl, legacyMode)
}
