// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {NameMappedObjects} from "mattermost-redux/types/utilities"

import {Channel} from "mattermost-redux/types/channels"

import {Team} from "./store/teams"


type Options = {
    singleline: boolean;
    atMentions: boolean;
    mentionHighlight: boolean;
    team: Team | null;
    channelNamesMap: NameMappedObjects<Channel>;
}

export type PostUtils = {
    formatText: (text: string, option: Options) => string;
    messageHtmlToComponent: (html: string) => React.ReactNode;
}

export type MattermostWindow = {
    PostUtils: PostUtils;
}


export const {
    formatText,
    messageHtmlToComponent,
} = (global as unknown as MattermostWindow).PostUtils ?? {}
