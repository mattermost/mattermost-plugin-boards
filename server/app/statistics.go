// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

func (a *App) GetUsedCardsCount() (int64, error) {
	return a.store.GetUsedCardsCount()
}
