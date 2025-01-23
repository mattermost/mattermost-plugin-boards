// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react'

import {FormattedMessage} from 'react-intl'

import {useMeasurePunchouts} from '../../tutorial_tour_tip/hooks'

import './add_comments.scss'
import {Utils} from '../../../utils'
import addComment from '../../../../static/comment.gif'

import {CardTourSteps, TOUR_CARD} from '../index'
import TourTipRenderer from '../tourTipRenderer/tourTipRenderer'

const AddCommentTourStep = (): JSX.Element | null => {
    const title = (
        <FormattedMessage
            id='OnboardingTour.AddComments.Title'
            defaultMessage='Add comments'
        />
    )
    const screen = (
        <FormattedMessage
            id='OnboardingTour.AddComments.Body'
            defaultMessage='You can comment on issues, and even @mention your fellow Mattermost users to get their attention.'
        />
    )

    const punchout = useMeasurePunchouts(['.CommentsList__new'], [])

    return (
        <TourTipRenderer
            key='AddCommentTourStep'
            requireCard={true}
            category={TOUR_CARD}
            step={CardTourSteps.ADD_COMMENTS}
            screen={screen}
            title={title}
            punchout={punchout}
            classname='AddCommentTourStep'
            telemetryTag='tourPoint2b'
            placement={'right-end'}
            imageURL={Utils.buildURL(addComment, true)}
            hideBackdrop={true}
        />
    )
}

export default AddCommentTourStep
