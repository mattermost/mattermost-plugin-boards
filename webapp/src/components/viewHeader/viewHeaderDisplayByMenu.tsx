// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.


import React from 'react'
import {FormattedMessage, useIntl} from 'react-intl'

import {DatePropertyType} from '../../properties/types'

import {IPropertyTemplate} from '../../blocks/board'
import {BoardView} from '../../blocks/boardView'
import mutator from '../../mutator'
import Button from '../../widgets/buttons/button'
import Menu from '../../widgets/menu'
import MenuWrapper from '../../widgets/menuWrapper'
import CheckIcon from '../../widgets/icons/check'

import propsRegistry from '../../properties'

type Props = {
    properties: readonly IPropertyTemplate[]
    activeView: BoardView
    dateDisplayPropertyName?: string
}

const ViewHeaderDisplayByMenu = (props: Props) => {
    const {properties, activeView, dateDisplayPropertyName} = props
    const intl = useIntl()

    const createdDateName = propsRegistry.get('createdTime').displayName(intl)

    const getDateProperties = (): IPropertyTemplate[] => {
        return properties?.filter((o: IPropertyTemplate) => propsRegistry.get(o.type) instanceof DatePropertyType)
    }

    return (
        <MenuWrapper>
            <Button>
                <FormattedMessage
                    id='ViewHeader.display-by'
                    defaultMessage='Display by: {property}'
                    values={{
                        property: (
                            <span
                                style={{color: 'rgb(var(--center-channel-color-rgb))'}}
                                id='displayByLabel'
                            >
                                {dateDisplayPropertyName || createdDateName}
                            </span>
                        ),
                    }}
                />
            </Button>
            <Menu>
                {getDateProperties().length > 0 && getDateProperties().map((date: IPropertyTemplate) => (
                    <Menu.Text
                        key={date.id}
                        id={date.id}
                        name={date.name}
                        rightIcon={activeView.fields.dateDisplayPropertyId === date.id ? <CheckIcon/> : undefined}
                        onClick={(id) => {
                            if (activeView.fields.dateDisplayPropertyId === id) {
                                return
                            }
                            mutator.changeViewDateDisplayPropertyId(activeView.boardId, activeView.id, activeView.fields.dateDisplayPropertyId, id)
                        }}
                    />
                ))}
                {getDateProperties().length === 0 &&
                    <Menu.Text
                        key={'createdDate'}
                        id={'createdDate'}
                        name={createdDateName}
                        rightIcon={<CheckIcon/>}
                        onClick={() => {}}
                    />
                }
            </Menu>
        </MenuWrapper>
    )
}

export default React.memo(ViewHeaderDisplayByMenu)
