// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-plugin-boards/server/model"

	"github.com/pkg/errors"
)

const (
	DaysPerMonth     = 30
	DaysPerWeek      = 7
	HoursPerDay      = 24
	MinutesPerHour   = 60
	SecondsPerMinute = 60
)

// IsValidReadToken validates the read token for a block.
func (a *App) IsValidReadToken(boardID string, readToken string) (bool, error) {
	return a.auth.IsValidReadToken(boardID, readToken)
}

// GetRegisteredUserCount returns the number of registered users.
func (a *App) GetRegisteredUserCount() (int, error) {
	return a.store.GetRegisteredUserCount()
}

// GetDailyActiveUsers returns the number of daily active users.
func (a *App) GetDailyActiveUsers() (int, error) {
	secondsAgo := int64(SecondsPerMinute * MinutesPerHour * HoursPerDay)
	return a.store.GetActiveUserCount(secondsAgo)
}

// GetWeeklyActiveUsers returns the number of weekly active users.
func (a *App) GetWeeklyActiveUsers() (int, error) {
	secondsAgo := int64(SecondsPerMinute * MinutesPerHour * HoursPerDay * DaysPerWeek)
	return a.store.GetActiveUserCount(secondsAgo)
}

// GetMonthlyActiveUsers returns the number of monthly active users.
func (a *App) GetMonthlyActiveUsers() (int, error) {
	secondsAgo := int64(SecondsPerMinute * MinutesPerHour * HoursPerDay * DaysPerMonth)
	return a.store.GetActiveUserCount(secondsAgo)
}

// GetUser gets an existing active user by id.
func (a *App) GetUser(id string) (*model.User, error) {
	if len(id) < 1 {
		return nil, errors.New("no user ID")
	}

	user, err := a.store.GetUserByID(id)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find user")
	}
	return user, nil
}

func (a *App) GetUsersList(userIDs []string) ([]*model.User, error) {
	if len(userIDs) == 0 {
		return nil, errors.New("No User IDs")
	}

	users, err := a.store.GetUsersList(userIDs, a.config.ShowEmailAddress, a.config.ShowFullName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find users")
	}
	return users, nil
}
