package app

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var mockUser = &model.User{
	ID:       utils.NewID(utils.IDTypeUser),
	Username: "testUsername",
	Email:    "testEmail",
	Password: "testPassword",
}

func TestGetUser(t *testing.T) {
	th, tearDown := SetupTestHelper(t)
	defer tearDown()

	testcases := []struct {
		title   string
		id      string
		isError bool
	}{
		{"fail, missing id", "", true},
		{"fail, invalid id", "badID", true},
		{"success", "goodID", false},
	}

	th.Store.EXPECT().GetUserByID("badID").Return(nil, errors.New("Bad Id"))
	th.Store.EXPECT().GetUserByID("goodID").Return(mockUser, nil)

	for _, test := range testcases {
		t.Run(test.title, func(t *testing.T) {
			token, err := th.App.GetUser(test.id)
			if test.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, token)
			}
		})
	}
}
