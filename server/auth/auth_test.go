// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package auth

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store/mockstore"
)

type TestHelper struct {
	Auth    *Auth
	Session model.Session
	Store   *mockstore.MockStore
}

func TestIsValidReadToken(t *testing.T) {
	// ToDo: reimplement

	// th := setupTestHelper(t)

	// validBlockID := "testBlockID"
	// mockContainer := store.Container{
	// 	TeamID: "testTeamID",
	// }
	// validReadToken := "testReadToken"
	// mockSharing := model.Sharing{
	// 	ID:      "testRootID",
	// 	Enabled: true,
	// 	Token:   validReadToken,
	// }

	// testcases := []struct {
	// 	title     string
	// 	container store.Container
	// 	blockID   string
	// 	readToken string
	// 	isError   bool
	// 	isSuccess bool
	// }{
	// 	{"fail, error GetRootID", mockContainer, "badBlock", "", true, false},
	// 	{"fail, rootID not found", mockContainer, "goodBlockID", "", false, false},
	// 	{"fail, sharing throws error", mockContainer, "goodBlockID2", "", true, false},
	// 	{"fail, bad readToken", mockContainer, validBlockID, "invalidReadToken", false, false},
	// 	{"success", mockContainer, validBlockID, validReadToken, false, true},
	// }

	// th.Store.EXPECT().GetRootID(gomock.Eq(mockContainer), "badBlock").Return("", errors.New("invalid block"))
	// th.Store.EXPECT().GetRootID(gomock.Eq(mockContainer), "goodBlockID").Return("rootNotFound", nil)
	// th.Store.EXPECT().GetRootID(gomock.Eq(mockContainer), "goodBlockID2").Return("rootError", nil)
	// th.Store.EXPECT().GetRootID(gomock.Eq(mockContainer), validBlockID).Return("testRootID", nil).Times(2)
	// th.Store.EXPECT().GetSharing(gomock.Eq(mockContainer), "rootNotFound").Return(nil, sql.ErrNoRows)
	// th.Store.EXPECT().GetSharing(gomock.Eq(mockContainer), "rootError").Return(nil, errors.New("another error"))
	// th.Store.EXPECT().GetSharing(gomock.Eq(mockContainer), "testRootID").Return(&mockSharing, nil).Times(2)

	// for _, test := range testcases {
	// 	t.Run(test.title, func(t *testing.T) {
	// 		success, err := th.Auth.IsValidReadToken(test.container, test.blockID, test.readToken)
	// 		if test.isError {
	// 			require.Error(t, err)
	// 		} else {
	// 			require.NoError(t, err)
	// 		}
	// 		if test.isSuccess {
	// 			require.True(t, success)
	// 		} else {
	// 			require.False(t, success)
	// 		}
	// 	})
	// }
}
