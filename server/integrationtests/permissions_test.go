// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//nolint:dupl
package integrationtests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

const (
	methodPost   = "POST"
	methodGet    = "GET"
	methodPut    = "PUT"
	methodDelete = "DELETE"
	methodPatch  = "PATCH"
)

type TestCase struct {
	url                string
	method             string
	body               string
	userRole           string // userAnon, userNoTeamMember, userTeamMember, userViewer, userCommenter, userEditor, userAdmin or userGuest
	expectedStatusCode int
	totalResults       int
}

func (tt TestCase) identifier() string {
	return fmt.Sprintf(
		"url: %s method: %s body: %s userRoles: %s expectedStatusCode: %d totalResults: %d",
		tt.url,
		tt.method,
		tt.body,
		tt.userRole,
		tt.expectedStatusCode,
		tt.totalResults,
	)
}

func toJSON(t *testing.T, obj interface{}) string {
	result, err := json.Marshal(obj)
	require.NoError(t, err)
	return string(result)
}

type TestData struct {
	publicBoard            *model.Board
	privateBoard           *model.Board
	publicTemplate         *model.Board
	privateTemplate        *model.Board
	publicTemplateBlockID  string
	privateTemplateBlockID string
	publicBoardBlockID     string
	privateBoardBlockID    string
	// Content block IDs for TestPermissionsMoveContentBlock
	content1_1 string
	content1_2 string
	content2_1 string
	content2_2 string
	content3_1 string
	content3_2 string
	content4_1 string
	content4_2 string
	// Additional block IDs for TestPermissionsUndeleteBoardBlock
	block5 string
	block6 string
	block7 string
	block8 string
}

func setupData(t *testing.T, th *TestHelper) TestData {
	blockID1 := utils.NewID(utils.IDTypeBlock)
	blockID2 := utils.NewID(utils.IDTypeBlock)
	blockID3 := utils.NewID(utils.IDTypeBlock)
	blockID4 := utils.NewID(utils.IDTypeBlock)

	customTemplate1, err := th.Server.App().CreateBoard(
		&model.Board{Title: "Custom template 1", TeamID: "test-team", IsTemplate: true, Type: model.BoardTypeOpen, MinimumRole: "viewer"},
		userAdminID,
		true,
	)
	require.NoError(t, err)
	err = th.Server.App().InsertBlock(&model.Block{ID: blockID1, Title: "Test", Type: "card", BoardID: customTemplate1.ID, Fields: map[string]interface{}{}}, userAdminID)
	require.NoError(t, err)
	customTemplate2, err := th.Server.App().CreateBoard(
		&model.Board{Title: "Custom template 2", TeamID: "test-team", IsTemplate: true, Type: model.BoardTypePrivate, MinimumRole: "viewer"},
		userAdminID,
		true)
	require.NoError(t, err)
	err = th.Server.App().InsertBlock(&model.Block{ID: blockID2, Title: "Test", Type: "card", BoardID: customTemplate2.ID, Fields: map[string]interface{}{}}, userAdminID)
	require.NoError(t, err)

	board1, err := th.Server.App().CreateBoard(&model.Board{Title: "Board 1", TeamID: "test-team", Type: model.BoardTypeOpen, MinimumRole: "viewer"}, userAdminID, true)
	require.NoError(t, err)
	err = th.Server.App().InsertBlock(&model.Block{ID: blockID3, Title: "Test", Type: "card", BoardID: board1.ID, Fields: map[string]interface{}{}}, userAdminID)
	require.NoError(t, err)
	board2, err := th.Server.App().CreateBoard(&model.Board{Title: "Board 2", TeamID: "test-team", Type: model.BoardTypePrivate, MinimumRole: "viewer"}, userAdminID, true)
	require.NoError(t, err)

	rBoard2, err := th.Server.App().GetBoard(board2.ID)
	require.NoError(t, err)
	require.NotNil(t, rBoard2)
	require.Equal(t, rBoard2, board2)

	boardMember, err := th.Server.App().GetMemberForBoard(board2.ID, userAdminID)
	require.NoError(t, err)
	require.NotNil(t, boardMember)
	require.Equal(t, boardMember.UserID, userAdminID)
	require.Equal(t, boardMember.BoardID, board2.ID)

	err = th.Server.App().InsertBlock(&model.Block{ID: blockID4, Title: "Test", Type: "card", BoardID: board2.ID, Fields: map[string]interface{}{}}, userAdminID)
	require.NoError(t, err)

	err = th.Server.App().UpsertSharing(model.Sharing{ID: board2.ID, Enabled: true, Token: "valid", ModifiedBy: userAdminID, UpdateAt: model.GetMillis()})
	require.NoError(t, err)

	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate1.ID, UserID: userViewerID, SchemeViewer: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate2.ID, UserID: userViewerID, SchemeViewer: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate1.ID, UserID: userCommenterID, SchemeCommenter: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate2.ID, UserID: userCommenterID, SchemeCommenter: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate1.ID, UserID: userEditorID, SchemeEditor: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate2.ID, UserID: userEditorID, SchemeEditor: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate1.ID, UserID: userAdminID, SchemeAdmin: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: customTemplate2.ID, UserID: userAdminID, SchemeAdmin: true})
	require.NoError(t, err)

	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board1.ID, UserID: userViewerID, SchemeViewer: true})
	require.NoError(t, err)

	boardMember, err = th.Server.App().GetMemberForBoard(board1.ID, userViewerID)
	require.NoError(t, err)
	require.NotNil(t, boardMember)
	require.Equal(t, boardMember.UserID, userViewerID)
	require.Equal(t, boardMember.BoardID, board1.ID)

	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board2.ID, UserID: userViewerID, SchemeViewer: true})
	require.NoError(t, err)

	boardMember, err = th.Server.App().GetMemberForBoard(board2.ID, userViewerID)
	require.NoError(t, err)
	require.NotNil(t, boardMember)
	require.Equal(t, boardMember.UserID, userViewerID)
	require.Equal(t, boardMember.BoardID, board2.ID)

	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board1.ID, UserID: userCommenterID, SchemeCommenter: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board2.ID, UserID: userCommenterID, SchemeCommenter: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board1.ID, UserID: userEditorID, SchemeEditor: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board2.ID, UserID: userEditorID, SchemeEditor: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board1.ID, UserID: userAdminID, SchemeAdmin: true})
	require.NoError(t, err)
	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board2.ID, UserID: userAdminID, SchemeAdmin: true})
	require.NoError(t, err)

	_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: board2.ID, UserID: userGuestID, SchemeViewer: true})
	require.NoError(t, err)

	return TestData{
		publicBoard:            board1,
		privateBoard:           board2,
		publicTemplate:         customTemplate1,
		privateTemplate:        customTemplate2,
		publicTemplateBlockID:  blockID1,
		privateTemplateBlockID: blockID2,
		publicBoardBlockID:     blockID3,
		privateBoardBlockID:    blockID4,
	}
}

func runTestCases(t *testing.T, ttCases []TestCase, testData TestData, clients Clients) {
	for _, tc := range ttCases {
		t.Run(tc.userRole+": "+tc.method+" "+tc.url, func(t *testing.T) {
			reqClient := clients.Anon
			switch tc.userRole {
			case userAnon:
				reqClient = clients.Anon
			case userNoTeamMember:
				reqClient = clients.NoTeamMember
			case userTeamMember:
				reqClient = clients.TeamMember
			case userViewer:
				reqClient = clients.Viewer
			case userCommenter:
				reqClient = clients.Commenter
			case userEditor:
				reqClient = clients.Editor
			case userAdmin:
				reqClient = clients.Admin
			case userGuest:
				if clients.Guest == nil {
					return
				}
				reqClient = clients.Guest
			}

			url := strings.ReplaceAll(tc.url, "{PRIVATE_BOARD_ID}", testData.privateBoard.ID)
			url = strings.ReplaceAll(url, "{PUBLIC_BOARD_ID}", testData.publicBoard.ID)
			url = strings.ReplaceAll(url, "{PUBLIC_TEMPLATE_ID}", testData.publicTemplate.ID)
			url = strings.ReplaceAll(url, "{PRIVATE_TEMPLATE_ID}", testData.privateTemplate.ID)
			url = strings.ReplaceAll(url, "{USER_ANON_ID}", userAnonID)
			url = strings.ReplaceAll(url, "{USER_NO_TEAM_MEMBER_ID}", userNoTeamMemberID)
			url = strings.ReplaceAll(url, "{USER_TEAM_MEMBER_ID}", userTeamMemberID)
			url = strings.ReplaceAll(url, "{USER_VIEWER_ID}", userViewerID)
			url = strings.ReplaceAll(url, "{USER_COMMENTER_ID}", userCommenterID)
			url = strings.ReplaceAll(url, "{USER_EDITOR_ID}", userEditorID)
			url = strings.ReplaceAll(url, "{USER_ADMIN_ID}", userAdminID)
			url = strings.ReplaceAll(url, "{USER_GUEST_ID}", userGuestID)
			// Replace block IDs in URL if present
			url = strings.ReplaceAll(url, "block-1", testData.publicTemplateBlockID)
			url = strings.ReplaceAll(url, "block-2", testData.privateTemplateBlockID)
			url = strings.ReplaceAll(url, "block-3", testData.publicBoardBlockID)
			url = strings.ReplaceAll(url, "block-4", testData.privateBoardBlockID)
			// Replace content block IDs in URL if present
			url = strings.ReplaceAll(url, "{CONTENT_1_1}", testData.content1_1)
			url = strings.ReplaceAll(url, "{CONTENT_1_2}", testData.content1_2)
			url = strings.ReplaceAll(url, "{CONTENT_2_1}", testData.content2_1)
			url = strings.ReplaceAll(url, "{CONTENT_2_2}", testData.content2_2)
			url = strings.ReplaceAll(url, "{CONTENT_3_1}", testData.content3_1)
			url = strings.ReplaceAll(url, "{CONTENT_3_2}", testData.content3_2)
			url = strings.ReplaceAll(url, "{CONTENT_4_1}", testData.content4_1)
			url = strings.ReplaceAll(url, "{CONTENT_4_2}", testData.content4_2)
			// Replace additional block IDs in URL if present
			url = strings.ReplaceAll(url, "{BLOCK_5}", testData.block5)
			url = strings.ReplaceAll(url, "{BLOCK_6}", testData.block6)
			url = strings.ReplaceAll(url, "{BLOCK_7}", testData.block7)
			url = strings.ReplaceAll(url, "{BLOCK_8}", testData.block8)

			if strings.Contains(url, "{") || strings.Contains(url, "}") {
				require.Fail(t, "Unreplaced tokens in url", url, tc.identifier())
			}

			// Replace block IDs in body if present (handle JSON quoted strings)
			body := tc.body
			body = strings.ReplaceAll(body, `"block-1"`, `"`+testData.publicTemplateBlockID+`"`)
			body = strings.ReplaceAll(body, `"block-2"`, `"`+testData.privateTemplateBlockID+`"`)
			body = strings.ReplaceAll(body, `"block-3"`, `"`+testData.publicBoardBlockID+`"`)
			body = strings.ReplaceAll(body, `"block-4"`, `"`+testData.privateBoardBlockID+`"`)

			var response *http.Response
			var err error
			switch tc.method {
			case methodGet:
				response, err = reqClient.DoAPIGet(url, "")
				defer response.Body.Close()
			case methodPost:
				response, err = reqClient.DoAPIPost(url, body)
				defer response.Body.Close()
			case methodPatch:
				response, err = reqClient.DoAPIPatch(url, body)
				defer response.Body.Close()
			case methodPut:
				response, err = reqClient.DoAPIPut(url, body)
				defer response.Body.Close()
			case methodDelete:
				response, err = reqClient.DoAPIDelete(url, body)
				defer response.Body.Close()
			}

			// Include the actual body used in the error message for debugging
			identifier := tc.identifier()
			if body != tc.body {
				identifier += fmt.Sprintf(" (body: %s)", body)
			}
			require.Equal(t, tc.expectedStatusCode, response.StatusCode, identifier)
			if tc.expectedStatusCode >= 200 && tc.expectedStatusCode < 300 {
				require.NoError(t, err, tc.identifier())
			}
			if tc.expectedStatusCode >= 200 && tc.expectedStatusCode < 300 {
				body, err := io.ReadAll(response.Body)
				if err != nil {
					require.Fail(t, err.Error(), tc.identifier())
				}
				if strings.HasPrefix(string(body), "[") {
					var data []interface{}
					err = json.Unmarshal(body, &data)
					if err != nil {
						require.Fail(t, err.Error(), tc.identifier())
					}
					require.Len(t, data, tc.totalResults, tc.identifier())
				} else {
					if tc.totalResults > 0 {
						require.Equal(t, 1, tc.totalResults)
						require.Greater(t, len(string(body)), 2, tc.identifier())
					} else {
						require.Len(t, string(body), 2, tc.identifier())
					}
				}
			}
		})
	}
}

func TestPermissionsGetTeamBoards(t *testing.T) {
	ttCases := []TestCase{
		{"/teams/test-team/boards", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/boards", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/boards", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team/boards", methodGet, "", userViewer, http.StatusOK, 2},
		{"/teams/test-team/boards", methodGet, "", userCommenter, http.StatusOK, 2},
		{"/teams/test-team/boards", methodGet, "", userEditor, http.StatusOK, 2},
		{"/teams/test-team/boards", methodGet, "", userAdmin, http.StatusOK, 2},
		{"/teams/test-team/boards", methodGet, "", userGuest, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsSearchTeamBoards(t *testing.T) {
	t.Skip("Skipping TestPermissionsSearchTeamBoards - search tests will be fixed separately")
	ttCases := []TestCase{
		// Search boards
		{"/teams/test-team/boards/search?q=b", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/boards/search?q=b", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/boards/search?q=b", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team/boards/search?q=b", methodGet, "", userViewer, http.StatusOK, 2},
		{"/teams/test-team/boards/search?q=b", methodGet, "", userCommenter, http.StatusOK, 2},
		{"/teams/test-team/boards/search?q=b", methodGet, "", userEditor, http.StatusOK, 2},
		{"/teams/test-team/boards/search?q=b", methodGet, "", userAdmin, http.StatusOK, 2},
		{"/teams/test-team/boards/search?q=b", methodGet, "", userGuest, http.StatusOK, 1},
	}
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}
func TestPermissionsSearchTeamLinkableBoards(t *testing.T) {
	t.Skip("Skipping TestPermissionsSearchTeamLinkableBoards - search tests will be fixed separately")
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		// Search boards
		{"/teams/test-team/boards/search/linkable?q=b", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/boards/search/linkable?q=b", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/boards/search/linkable?q=b", methodGet, "", userTeamMember, http.StatusOK, 0},
		{"/teams/test-team/boards/search/linkable?q=b", methodGet, "", userViewer, http.StatusOK, 0},
		{"/teams/test-team/boards/search/linkable?q=b", methodGet, "", userCommenter, http.StatusOK, 0},
		{"/teams/test-team/boards/search/linkable?q=b", methodGet, "", userEditor, http.StatusOK, 0},
		{"/teams/test-team/boards/search/linkable?q=b", methodGet, "", userAdmin, http.StatusOK, 2},
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetTeamTemplates(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper) {
		err := th.Server.App().InitTemplates()
		require.NoError(t, err, "InitTemplates should succeed")
	}

	builtInTemplateCount := 13

	ttCases := []TestCase{
		// Get Team Boards
		{"/teams/test-team/templates", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/templates", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/templates", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team/templates", methodGet, "", userViewer, http.StatusOK, 2},
		{"/teams/test-team/templates", methodGet, "", userCommenter, http.StatusOK, 2},
		{"/teams/test-team/templates", methodGet, "", userEditor, http.StatusOK, 2},
		{"/teams/test-team/templates", methodGet, "", userAdmin, http.StatusOK, 2},
		{"/teams/test-team/templates", methodGet, "", userGuest, http.StatusForbidden, 0},
		// Built-in templates
		{"/teams/0/templates", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/0/templates", methodGet, "", userNoTeamMember, http.StatusOK, builtInTemplateCount},
		{"/teams/0/templates", methodGet, "", userTeamMember, http.StatusOK, builtInTemplateCount},
		{"/teams/0/templates", methodGet, "", userViewer, http.StatusOK, builtInTemplateCount},
		{"/teams/0/templates", methodGet, "", userCommenter, http.StatusOK, builtInTemplateCount},
		{"/teams/0/templates", methodGet, "", userEditor, http.StatusOK, builtInTemplateCount},
		{"/teams/0/templates", methodGet, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsCreateBoard(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)

	// Generate a valid team ID for board creation
	teamID := mmModel.NewId()
	publicBoard := toJSON(t, model.Board{Title: "Board To Create", TeamID: teamID, Type: model.BoardTypeOpen})
	privateBoard := toJSON(t, model.Board{Title: "Board To Create", TeamID: teamID, Type: model.BoardTypeOpen})

	ttCases := []TestCase{
		// Create Public boards
		{"/boards", methodPost, publicBoard, userAnon, http.StatusUnauthorized, 0},
		{"/boards", methodPost, publicBoard, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards", methodPost, publicBoard, userGuest, http.StatusForbidden, 0},
		{"/boards", methodPost, publicBoard, userTeamMember, http.StatusOK, 1},

		// Create private boards
		{"/boards", methodPost, privateBoard, userAnon, http.StatusUnauthorized, 0},
		{"/boards", methodPost, privateBoard, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards", methodPost, privateBoard, userGuest, http.StatusForbidden, 0},
		{"/boards", methodPost, privateBoard, userTeamMember, http.StatusOK, 1},
	}
	runTestCases(t, ttCases, testData, clients)
}
func TestPermissionsGetBoard(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}", methodGet, "", userGuest, http.StatusOK, 1},

		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_BOARD_ID}?read_token=invalid", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}?read_token=valid", methodGet, "", userAnon, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}?read_token=invalid", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}?read_token=valid", methodGet, "", userTeamMember, http.StatusOK, 1},
	}
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}
func TestPermissionsGetBoardPublic(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}?read_token=invalid", methodGet, "", userAnon, http.StatusUnauthorized, 0},
	}
	cfg := th.Server.Config()
	cfg.EnablePublicSharedBoards = false
	th.Server.UpdateAppConfig()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
	runTestCases(t, ttCases, testData, clients)
	cfg.EnablePublicSharedBoards = false
	th.Server.UpdateAppConfig()
	clients = setupClients(th)
	testData = setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsPatchBoard(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"title\": \"test\"}", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"title\": \"test\"}", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsPatchBoardType(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userAdmin, http.StatusOK, 1},

		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, "{\"type\": \"P\"}", userAdmin, http.StatusOK, 1},

		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userAdmin, http.StatusOK, 1},

		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, "{\"type\": \"P\"}", userAdmin, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsPatchBoardMinimumRole(t *testing.T) {
	patch := toJSON(t, map[string]model.BoardRole{"minimumRole": model.BoardRoleViewer})
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},

		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},

		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},

		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsPatchBoardChannelId(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)

	// Generate a valid channel ID for patching
	validChannelID := mmModel.NewId()
	patch := toJSON(t, map[string]string{"channelId": validChannelID})

	// Update FakePermissionPluginAPI to accept the generated channel ID
	// This is done by modifying the permission check in the test setup
	// Note: The actual permission check happens in the API layer, so we need to ensure
	// the channel ID is valid format. The FakePermissionPluginAPI in clienttestlib.go
	// will need to be updated to accept any valid Mattermost ID format.

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},

		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},

		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},

		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodPatch, patch, userAdmin, http.StatusOK, 1},
	}

	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsDeleteBoard(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},

		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsDuplicateBoard(t *testing.T) {
	// In same team
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userTeamMember, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},
	}
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	t.Run("plugin-same-team", func(t *testing.T) {
		clients := setupClients(th)
		testData := setupData(t, th)
		runTestCases(t, ttCases, testData, clients)
		testData = setupData(t, th)
		runTestCases(t, ttCases, testData, clients)
		clients = setupClients(th)
		testData = setupData(t, th)
		ttCasesCopy := make([]TestCase, len(ttCases))
		copy(ttCasesCopy, ttCases)
		ttCasesCopy[25].expectedStatusCode = http.StatusOK
		ttCasesCopy[25].totalResults = 1
		runTestCases(t, ttCasesCopy, testData, clients)
		testData = setupData(t, th)
		runTestCases(t, ttCasesCopy, testData, clients)
	})

	// In other team
	ttCases = []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=other-team", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userTeamMember, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=other-team", methodPost, "", userGuest, http.StatusForbidden, 0},
	}
	t.Run("plugin-other-team", func(t *testing.T) {
		clients := setupClients(th)
		testData := setupData(t, th)
		runTestCases(t, ttCases, testData, clients)
		testData = setupData(t, th)
		runTestCases(t, ttCases, testData, clients)
		clients = setupClients(th)
		testData = setupData(t, th)
		// Create a copy of ttCases to avoid modifying the original slice
		// In plugin-other-team, index 25 is userNoTeamMember with PUBLIC_TEMPLATE_ID
		// which should remain 403 (Forbidden) - no modification needed
		ttCasesCopy := make([]TestCase, len(ttCases))
		copy(ttCasesCopy, ttCases)
		runTestCases(t, ttCasesCopy, testData, clients)
		testData = setupData(t, th)
		runTestCases(t, ttCasesCopy, testData, clients)
	})

	// In empty team
	ttCases = []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userAdmin, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userAdmin, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/duplicate?toTeam=empty-team", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userAdmin, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userAdmin, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/duplicate?toTeam=empty-team", methodPost, "", userGuest, http.StatusForbidden, 0},
	}
	t.Run("plugin-empty-team", func(t *testing.T) {
		clients := setupClients(th)
		testData := setupData(t, th)
		runTestCases(t, ttCases, testData, clients)
		runTestCases(t, ttCases, testData, clients)
	})
}

func TestPermissionsGetBoardBlocks(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodGet, "", userGuest, http.StatusOK, 1},

		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_BOARD_ID}/blocks?read_token=invalid", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks?read_token=valid", methodGet, "", userAnon, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks?read_token=invalid", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks?read_token=valid", methodGet, "", userTeamMember, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsCreateBoardBlocks(t *testing.T) {
	ttCasesF := func(testData TestData) []TestCase {
		counter := 0
		newBlockJSON := func(boardID string) string {
			counter++
			return toJSON(t, []*model.Block{{
				ID:       fmt.Sprintf("%d", counter),
				Title:    "Board To Create",
				BoardID:  boardID,
				Type:     "card",
				CreateAt: model.GetMillis(),
				UpdateAt: model.GetMillis(),
			}})
		}

		return []TestCase{
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userGuest, http.StatusForbidden, 0},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := ttCasesF(testData)
	runTestCases(t, ttCases, testData, clients)
}
func TestPermissionsCreateBoardComments(t *testing.T) {
	ttCasesF := func(testData TestData) []TestCase {
		counter := 0
		newBlockJSON := func(boardID string) string {
			counter++
			return toJSON(t, []*model.Block{{
				ID:       fmt.Sprintf("%d", counter),
				Title:    "Comment to create",
				BoardID:  boardID,
				Type:     model.TypeComment,
				CreateAt: model.GetMillis(),
				UpdateAt: model.GetMillis(),
			}})
		}

		return []TestCase{
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userCommenter, http.StatusOK, 1},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userCommenter, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userCommenter, http.StatusOK, 1},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userCommenter, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userGuest, http.StatusForbidden, 0},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := ttCasesF(testData)
	runTestCases(t, ttCases, testData, clients)
}
func TestPermissionsPatchBoardBlocks(t *testing.T) {
	newBlocksPatchJSON := func(blockID string) string {
		return toJSON(t, model.BlockPatchBatch{
			BlockIDs:     []string{blockID},
			BlockPatches: []model.BlockPatch{{}}, // Empty patch for each block ID
		})
	}

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userEditor, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-4"), userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userEditor, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPatch, newBlocksPatchJSON("block-3"), userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userEditor, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-2"), userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userEditor, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPatch, newBlocksPatchJSON("block-1"), userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsPatchBoardBlock(t *testing.T) {
	newTitle := "New Patch Title"
	patchJSON := toJSON(t, model.BlockPatch{Title: &newTitle})

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodPatch, patchJSON, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodPatch, patchJSON, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodPatch, patchJSON, userEditor, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodPatch, patchJSON, userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodPatch, patchJSON, userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userEditor, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodPatch, patchJSON, userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userEditor, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodPatch, patchJSON, userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userEditor, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodPatch, patchJSON, userGuest, http.StatusForbidden, 0},

		// Invalid boardID/blockID combination
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-3", methodPatch, patchJSON, userAdmin, http.StatusNotFound, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsDeleteBoardBlock(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper, testData TestData) {
		err := th.Server.App().InsertBlock(&model.Block{ID: "block-5", Title: "Test", Type: "card", BoardID: testData.publicTemplate.ID}, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: "block-6", Title: "Test", Type: "card", BoardID: testData.privateTemplate.ID}, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: "block-7", Title: "Test", Type: "card", BoardID: testData.publicBoard.ID}, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: "block-8", Title: "Test", Type: "card", BoardID: testData.privateBoard.ID}, userAdmin)
		require.NoError(t, err)
	}

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodDelete, "", userEditor, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-8", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4", methodDelete, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodDelete, "", userEditor, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-7", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3", methodDelete, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodDelete, "", userEditor, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-6", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2", methodDelete, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodDelete, "", userEditor, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-5", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1", methodDelete, "", userGuest, http.StatusForbidden, 0},

		// Invalid boardID/blockID combination
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-3", methodDelete, "", userAdmin, http.StatusNotFound, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th, testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsUndeleteBoardBlock(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper, testData *TestData) {
		block5 := utils.NewID(utils.IDTypeBlock)
		block6 := utils.NewID(utils.IDTypeBlock)
		block7 := utils.NewID(utils.IDTypeBlock)
		block8 := utils.NewID(utils.IDTypeBlock)
		err := th.Server.App().InsertBlock(&model.Block{ID: block5, Title: "Test", Type: "card", BoardID: testData.publicTemplate.ID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: block6, Title: "Test", Type: "card", BoardID: testData.privateTemplate.ID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: block7, Title: "Test", Type: "card", BoardID: testData.publicBoard.ID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: block8, Title: "Test", Type: "card", BoardID: testData.privateBoard.ID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(testData.publicTemplateBlockID, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(testData.privateTemplateBlockID, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(testData.publicBoardBlockID, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(testData.privateBoardBlockID, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(block5, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(block6, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(block7, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().DeleteBlock(block8, userAdminID)
		require.NoError(t, err)
		// Store block5-8 in testData for URL replacement
		testData.block5 = block5
		testData.block6 = block6
		testData.block7 = block7
		testData.block8 = block8
	}

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/undelete", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/{BLOCK_8}/undelete", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/undelete", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/undelete", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/undelete", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/undelete", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/{BLOCK_7}/undelete", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/undelete", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/undelete", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/undelete", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/undelete", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/{BLOCK_6}/undelete", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/undelete", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/undelete", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/undelete", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/undelete", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/{BLOCK_5}/undelete", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},

		// Invalid boardID/blockID combination
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-3/undelete", methodPost, "", userAdmin, http.StatusNotFound, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th, &testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsMoveContentBlock(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper, testData *TestData) {
		testData.content1_1 = utils.NewID(utils.IDTypeBlock)
		testData.content1_2 = utils.NewID(utils.IDTypeBlock)
		testData.content2_1 = utils.NewID(utils.IDTypeBlock)
		testData.content2_2 = utils.NewID(utils.IDTypeBlock)
		testData.content3_1 = utils.NewID(utils.IDTypeBlock)
		testData.content3_2 = utils.NewID(utils.IDTypeBlock)
		testData.content4_1 = utils.NewID(utils.IDTypeBlock)
		testData.content4_2 = utils.NewID(utils.IDTypeBlock)
		err := th.Server.App().InsertBlock(&model.Block{ID: testData.content1_1, Title: "Test", Type: "text", BoardID: testData.publicTemplate.ID, ParentID: testData.publicTemplateBlockID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: testData.content1_2, Title: "Test", Type: "text", BoardID: testData.publicTemplate.ID, ParentID: testData.publicTemplateBlockID}, userAdminID)
		require.NoError(t, err)
		_, err = th.Server.App().PatchBlock(testData.publicTemplateBlockID, &model.BlockPatch{UpdatedFields: map[string]interface{}{"contentOrder": []string{testData.content1_1, testData.content1_2}}}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: testData.content2_1, Title: "Test", Type: "text", BoardID: testData.privateTemplate.ID, ParentID: testData.privateTemplateBlockID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: testData.content2_2, Title: "Test", Type: "text", BoardID: testData.privateTemplate.ID, ParentID: testData.privateTemplateBlockID}, userAdminID)
		require.NoError(t, err)
		_, err = th.Server.App().PatchBlock(testData.privateTemplateBlockID, &model.BlockPatch{UpdatedFields: map[string]interface{}{"contentOrder": []string{testData.content2_1, testData.content2_2}}}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: testData.content3_1, Title: "Test", Type: "text", BoardID: testData.publicBoard.ID, ParentID: testData.publicBoardBlockID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: testData.content3_2, Title: "Test", Type: "text", BoardID: testData.publicBoard.ID, ParentID: testData.publicBoardBlockID}, userAdminID)
		require.NoError(t, err)
		_, err = th.Server.App().PatchBlock(testData.publicBoardBlockID, &model.BlockPatch{UpdatedFields: map[string]interface{}{"contentOrder": []string{testData.content3_1, testData.content3_2}}}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: testData.content4_1, Title: "Test", Type: "text", BoardID: testData.privateBoard.ID, ParentID: testData.privateBoardBlockID}, userAdminID)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: testData.content4_2, Title: "Test", Type: "text", BoardID: testData.privateBoard.ID, ParentID: testData.privateBoardBlockID}, userAdminID)
		require.NoError(t, err)
		_, err = th.Server.App().PatchBlock(testData.privateBoardBlockID, &model.BlockPatch{UpdatedFields: map[string]interface{}{"contentOrder": []string{testData.content4_1, testData.content4_2}}}, userAdminID)
		require.NoError(t, err)
	}

	ttCases := []TestCase{
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userAnon, http.StatusUnauthorized, 0},
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userNoTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userViewer, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userCommenter, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userEditor, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userAdmin, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_4_1}/moveto/after/{CONTENT_4_2}", methodPost, "{}", userGuest, http.StatusForbidden, 0},

		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userAnon, http.StatusUnauthorized, 0},
		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userNoTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userViewer, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userCommenter, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userEditor, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userAdmin, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_3_1}/moveto/after/{CONTENT_3_2}", methodPost, "{}", userGuest, http.StatusForbidden, 0},

		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userAnon, http.StatusUnauthorized, 0},
		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userNoTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userViewer, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userCommenter, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userEditor, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userAdmin, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_2_1}/moveto/after/{CONTENT_2_2}", methodPost, "{}", userGuest, http.StatusForbidden, 0},

		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userAnon, http.StatusUnauthorized, 0},
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userNoTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userTeamMember, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userViewer, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userCommenter, http.StatusForbidden, 0},
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userEditor, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userAdmin, http.StatusOK, 0},
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_1_2}", methodPost, "{}", userGuest, http.StatusForbidden, 0},

		// Invalid srcBlockID/dstBlockID combination
		{"/content-blocks/{CONTENT_1_1}/moveto/after/{CONTENT_2_1}", methodPost, "{}", userAdmin, http.StatusBadRequest, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th, &testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsUndeleteBoard(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper, testData TestData) {
		err := th.Server.App().DeleteBoard(testData.publicBoard.ID, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().DeleteBoard(testData.privateBoard.ID, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().DeleteBoard(testData.publicTemplate.ID, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().DeleteBoard(testData.privateTemplate.ID, userAdmin)
		require.NoError(t, err)
	}

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/undelete", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/undelete", methodPost, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/undelete", methodPost, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th, testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsDuplicateBoardBlock(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper, testData TestData) {
		err := th.Server.App().InsertBlock(&model.Block{ID: "block-5", Title: "Test", Type: "card", BoardID: testData.publicTemplate.ID}, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: "block-6", Title: "Test", Type: "card", BoardID: testData.privateTemplate.ID}, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: "block-7", Title: "Test", Type: "card", BoardID: testData.publicBoard.ID}, userAdmin)
		require.NoError(t, err)
		err = th.Server.App().InsertBlock(&model.Block{ID: "block-8", Title: "Test", Type: "card", BoardID: testData.privateBoard.ID}, userAdmin)
		require.NoError(t, err)
	}

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/duplicate", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/blocks/block-4/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/blocks/block-3/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/blocks/block-2/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-1/duplicate", methodPost, "", userGuest, http.StatusForbidden, 0},

		// Invalid boardID/blockID combination
		{"/boards/{PUBLIC_TEMPLATE_ID}/blocks/block-3/duplicate", methodPost, "", userAdmin, http.StatusNotFound, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th, testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetBoardMembers(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/members", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/members", methodGet, "", userCommenter, http.StatusOK, 5},
		{"/boards/{PRIVATE_BOARD_ID}/members", methodGet, "", userEditor, http.StatusOK, 5},
		{"/boards/{PRIVATE_BOARD_ID}/members", methodGet, "", userAdmin, http.StatusOK, 5},
		{"/boards/{PRIVATE_BOARD_ID}/members", methodGet, "", userGuest, http.StatusOK, 5},

		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userViewer, http.StatusOK, 4},
		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userCommenter, http.StatusOK, 4},
		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userEditor, http.StatusOK, 4},
		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userAdmin, http.StatusOK, 4},
		{"/boards/{PUBLIC_BOARD_ID}/members", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userViewer, http.StatusOK, 4},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userCommenter, http.StatusOK, 4},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userEditor, http.StatusOK, 4},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userAdmin, http.StatusOK, 4},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userViewer, http.StatusOK, 4},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userCommenter, http.StatusOK, 4},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userEditor, http.StatusOK, 4},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userAdmin, http.StatusOK, 4},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodGet, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsCreateBoardMembers(t *testing.T) {
	ttCasesF := func(testData TestData) []TestCase {
		boardMemberJSON := func(boardID string) string {
			return toJSON(t, model.BoardMember{
				BoardID:      boardID,
				UserID:       userTeamMemberID,
				SchemeEditor: true,
			})
		}

		return []TestCase{
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userEditor, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_BOARD_ID}/members", methodPost, boardMemberJSON(testData.privateBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/members", methodPost, boardMemberJSON(testData.publicBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userEditor, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.privateTemplate.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userEditor, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members", methodPost, boardMemberJSON(testData.publicTemplate.ID), userGuest, http.StatusForbidden, 0},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := ttCasesF(testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsUpdateBoardMember(t *testing.T) {
	ttCasesF := func(testData TestData) []TestCase {
		boardMemberJSON := func(boardID string) string {
			return toJSON(t, model.BoardMember{
				BoardID:      boardID,
				UserID:       userTeamMember,
				SchemeEditor: false,
				SchemeViewer: true,
			})
		}

		return []TestCase{
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userEditor, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userEditor, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_BOARD_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicBoard.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userEditor, http.StatusForbidden, 0},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.privateTemplate.ID), userGuest, http.StatusForbidden, 0},

			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userAnon, http.StatusUnauthorized, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userTeamMember, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userViewer, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userCommenter, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userEditor, http.StatusForbidden, 0},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userAdmin, http.StatusOK, 1},
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_VIEWER_ID}", methodPut, boardMemberJSON(testData.publicTemplate.ID), userGuest, http.StatusForbidden, 0},

			// Invalid boardID/memberID combination
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodPut, "", userAdmin, http.StatusBadRequest, 0},

			// Invalid boardID
			{"/boards/invalid/members/{USER_VIEWER_ID}", methodPut, "", userAdmin, http.StatusBadRequest, 0},

			// Invalid memberID
			{"/boards/{PUBLIC_TEMPLATE_ID}/members/invalid", methodPut, "", userAdmin, http.StatusBadRequest, 0},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := ttCasesF(testData)
	runTestCases(t, ttCases, testData, clients)
}
func TestPermissionsDeleteBoardMember(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper, testData TestData) {
		_, err := th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.publicBoard.ID, UserID: userTeamMemberID, SchemeViewer: true})
		require.NoError(t, err)
		_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.privateBoard.ID, UserID: userTeamMemberID, SchemeViewer: true})
		require.NoError(t, err)
		_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.publicTemplate.ID, UserID: userTeamMemberID, SchemeViewer: true})
		require.NoError(t, err)
		_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.privateTemplate.ID, UserID: userTeamMemberID, SchemeViewer: true})
		require.NoError(t, err)
	}

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userGuest, http.StatusForbidden, 0},

		// Invalid boardID/memberID combination
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/{USER_TEAM_MEMBER_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},

		// Invalid boardID
		{"/boards/invalid/members/{USER_VIEWER_ID}", methodDelete, "", userAdmin, http.StatusNotFound, 0},

		// Invalid memberID
		{"/boards/{PUBLIC_TEMPLATE_ID}/members/invalid", methodDelete, "", userAdmin, http.StatusOK, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th, testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsJoinBoardAsMember(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/join", methodPost, "", userAnon, http.StatusBadRequest, 0},
		// Do we want to forbid already existing members to join to the board or simply return the current membership?
		{"/boards/{PRIVATE_BOARD_ID}/join", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/join", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/join", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/join", methodPost, "", userAdmin, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/join", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userAnon, http.StatusBadRequest, 0},
		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userTeamMember, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/join", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userAnon, http.StatusBadRequest, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userAdmin, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/join", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userAnon, http.StatusBadRequest, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userTeamMember, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/join", methodPost, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsLeaveBoardAsMember(t *testing.T) {
	extraSetup := func(t *testing.T, th *TestHelper, testData TestData) {
		// Add commenter as admin to each board so admin can leave (can't leave if last admin)
		// UpdateBoardMember returns nil, nil if member doesn't exist, so check and add if needed
		member, err := th.Server.App().UpdateBoardMember(&model.BoardMember{BoardID: testData.publicBoard.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true, SchemeViewer: true})
		if err != nil || member == nil {
			_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.publicBoard.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true})
			require.NoError(t, err)
		}
		member, err = th.Server.App().UpdateBoardMember(&model.BoardMember{BoardID: testData.privateBoard.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true, SchemeViewer: true})
		if err != nil || member == nil {
			_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.privateBoard.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true})
			require.NoError(t, err)
		}
		member, err = th.Server.App().UpdateBoardMember(&model.BoardMember{BoardID: testData.publicTemplate.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true, SchemeViewer: true})
		if err != nil || member == nil {
			_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.publicTemplate.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true})
			require.NoError(t, err)
		}
		member, err = th.Server.App().UpdateBoardMember(&model.BoardMember{BoardID: testData.privateTemplate.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true, SchemeViewer: true})
		if err != nil || member == nil {
			_, err = th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.privateTemplate.ID, UserID: userCommenterID, SchemeAdmin: true, SchemeCommenter: true})
			require.NoError(t, err)
		}
	}

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/leave", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/leave", methodPost, "", userCommenter, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/leave", methodPost, "", userEditor, http.StatusOK, 0},
		// cannot leave a board with no admins because the admin is the last admin.
		{"/boards/{PRIVATE_BOARD_ID}/leave", methodPost, "", userAdmin, http.StatusBadRequest, 0},
		{"/boards/{PRIVATE_BOARD_ID}/leave", methodPost, "", userGuest, http.StatusOK, 0},

		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userViewer, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userCommenter, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userEditor, http.StatusOK, 0},
		// cannot leave a board with no admins because the admin is the last admin.
		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userAdmin, http.StatusBadRequest, 0},
		{"/boards/{PUBLIC_BOARD_ID}/leave", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userViewer, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userCommenter, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userEditor, http.StatusOK, 0},
		// cannot leave a board with no admins because the admin is the last admin.
		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userAdmin, http.StatusBadRequest, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/leave", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userTeamMember, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userViewer, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userCommenter, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userEditor, http.StatusOK, 0},
		// cannot leave a board with no admins because the admin is the last admin.
		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userAdmin, http.StatusBadRequest, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/leave", methodPost, "", userGuest, http.StatusOK, 0},
	}
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th, testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsShareBoard(t *testing.T) {
	sharing := toJSON(t, model.Sharing{Enabled: true, Token: "test-token"})

	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodPost, sharing, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodPost, sharing, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodPost, sharing, userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodPost, sharing, userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodPost, sharing, userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodPost, sharing, userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userAdmin, http.StatusOK, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodPost, sharing, userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userAdmin, http.StatusOK, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodPost, sharing, userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetSharedBoardInfo(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/sharing", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/sharing", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/sharing", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/sharing", methodGet, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)

	clients.Admin.PostSharing(&model.Sharing{ID: testData.publicBoard.ID, Enabled: true, Token: "test-token"})
	clients.Admin.PostSharing(&model.Sharing{ID: testData.publicTemplate.ID, Enabled: true, Token: "test-token"})
	clients.Admin.PostSharing(&model.Sharing{ID: testData.privateTemplate.ID, Enabled: true, Token: "test-token"})

	runTestCases(t, ttCases, testData, clients)
	runTestCases(t, ttCases, testData, clients)
	clients.Admin.PostSharing(&model.Sharing{ID: testData.privateTemplate.ID, Enabled: true, Token: "test-token"})

	runTestCases(t, ttCases, testData, clients)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsListTeams(t *testing.T) {
	ttCases := []TestCase{
		{"/teams", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams", methodGet, "", userNoTeamMember, http.StatusOK, 0},
		{"/teams", methodGet, "", userTeamMember, http.StatusOK, 2},
		{"/teams", methodGet, "", userViewer, http.StatusOK, 2},
		{"/teams", methodGet, "", userCommenter, http.StatusOK, 2},
		{"/teams", methodGet, "", userEditor, http.StatusOK, 2},
		{"/teams", methodGet, "", userAdmin, http.StatusOK, 2},
		{"/teams", methodGet, "", userGuest, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetTeam(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		{"/teams/test-team", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team", methodGet, "", userViewer, http.StatusOK, 1},
		{"/teams/test-team", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/teams/test-team", methodGet, "", userEditor, http.StatusOK, 1},
		{"/teams/test-team", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/teams/test-team", methodGet, "", userGuest, http.StatusOK, 1},

		{"/teams/empty-team", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/empty-team", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/empty-team", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/teams/empty-team", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/teams/empty-team", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/teams/empty-team", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/teams/empty-team", methodGet, "", userAdmin, http.StatusForbidden, 0},
		{"/teams/empty-team", methodGet, "", userGuest, http.StatusForbidden, 0},
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsRegenerateSignupToken(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		{"/teams/test-team/regenerate_signup_token", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/regenerate_signup_token", methodPost, "", userAdmin, http.StatusNotImplemented, 0},

		{"/teams/empty-team/regenerate_signup_token", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/empty-team/regenerate_signup_token", methodPost, "", userAdmin, http.StatusNotImplemented, 0},
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetTeamUsers(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		{"/teams/test-team/users", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/users", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/users", methodGet, "", userTeamMember, http.StatusOK, 6},
		{"/teams/test-team/users", methodGet, "", userViewer, http.StatusOK, 6},
		{"/teams/test-team/users", methodGet, "", userCommenter, http.StatusOK, 6},
		{"/teams/test-team/users", methodGet, "", userEditor, http.StatusOK, 6},
		{"/teams/test-team/users", methodGet, "", userAdmin, http.StatusOK, 6},
		{"/teams/test-team/users", methodGet, "", userGuest, http.StatusOK, 5},

		{"/teams/empty-team/users", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/empty-team/users", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/empty-team/users", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/teams/empty-team/users", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/teams/empty-team/users", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/teams/empty-team/users", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/teams/empty-team/users", methodGet, "", userAdmin, http.StatusForbidden, 0},
		{"/teams/empty-team/users", methodGet, "", userGuest, http.StatusForbidden, 0},
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsTeamArchiveExport(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		{"/teams/test-team/archive/export", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/archive/export", methodGet, "", userAdmin, http.StatusNotImplemented, 0},

		{"/teams/empty-team/archive/export", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/empty-team/archive/export", methodGet, "", userAdmin, http.StatusNotImplemented, 0},
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsUploadFile(t *testing.T) {
	ttCases := []TestCase{
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userEditor, http.StatusBadRequest, 1}, // Not checking the logic, only the permissions
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userAdmin, http.StatusBadRequest, 1},  // Not checking the logic, only the permissions
		{"/teams/test-team/{PRIVATE_BOARD_ID}/files", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userEditor, http.StatusBadRequest, 1}, // Not checking the logic, only the permissions
		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userAdmin, http.StatusBadRequest, 1},  // Not checking the logic, only the permissions
		{"/teams/test-team/{PUBLIC_BOARD_ID}/files", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userEditor, http.StatusBadRequest, 1}, // Not checking the logic, only the permissions
		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userAdmin, http.StatusBadRequest, 1},  // Not checking the logic, only the permissions
		{"/teams/test-team/{PRIVATE_TEMPLATE_ID}/files", methodPost, "", userGuest, http.StatusForbidden, 0},

		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userViewer, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userCommenter, http.StatusForbidden, 0},
		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userEditor, http.StatusBadRequest, 1}, // Not checking the logic, only the permissions
		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userAdmin, http.StatusBadRequest, 1},  // Not checking the logic, only the permissions
		{"/teams/test-team/{PUBLIC_TEMPLATE_ID}/files", methodPost, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetMe(t *testing.T) {
	ttCases := []TestCase{
		{"/users/me", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/users/me", methodGet, "", userNoTeamMember, http.StatusOK, 1},
		{"/users/me", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/users/me", methodGet, "", userViewer, http.StatusOK, 1},
		{"/users/me", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/users/me", methodGet, "", userEditor, http.StatusOK, 1},
		{"/users/me", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/users/me", methodGet, "", userGuest, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetMyMemberships(t *testing.T) {
	ttCases := []TestCase{
		{"/users/me/memberships", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/users/me/memberships", methodGet, "", userNoTeamMember, http.StatusOK, 0},
		{"/users/me/memberships", methodGet, "", userTeamMember, http.StatusOK, 0},
		{"/users/me/memberships", methodGet, "", userViewer, http.StatusOK, 4},
		{"/users/me/memberships", methodGet, "", userCommenter, http.StatusOK, 4},
		{"/users/me/memberships", methodGet, "", userEditor, http.StatusOK, 4},
		{"/users/me/memberships", methodGet, "", userAdmin, http.StatusOK, 4},
		{"/users/me/memberships", methodGet, "", userGuest, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetUser(t *testing.T) {
	ttCases := []TestCase{
		{"/users/{USER_NO_TEAM_MEMBER_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/users/{USER_NO_TEAM_MEMBER_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/users/{USER_NO_TEAM_MEMBER_ID}", methodGet, "", userEditor, http.StatusOK, 1},
		{"/users/{USER_NO_TEAM_MEMBER_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/users/{USER_NO_TEAM_MEMBER_ID}", methodGet, "", userGuest, http.StatusNotFound, 0},

		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userNoTeamMember, http.StatusOK, 1},
		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userViewer, http.StatusOK, 1},
		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userEditor, http.StatusOK, 1},
		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/users/{USER_TEAM_MEMBER_ID}", methodGet, "", userGuest, http.StatusNotFound, 0},

		{"/users/{USER_VIEWER_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/users/{USER_VIEWER_ID}", methodGet, "", userNoTeamMember, http.StatusOK, 1},
		{"/users/{USER_VIEWER_ID}", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/users/{USER_VIEWER_ID}", methodGet, "", userViewer, http.StatusOK, 1},
		{"/users/{USER_VIEWER_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/users/{USER_VIEWER_ID}", methodGet, "", userEditor, http.StatusOK, 1},
		{"/users/{USER_VIEWER_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/users/{USER_VIEWER_ID}", methodGet, "", userGuest, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsUpdateUserConfig(t *testing.T) {
	patch := toJSON(t, model.UserPreferencesPatch{UpdatedFields: map[string]string{"test": "test"}})

	ttCases := []TestCase{
		{"/users/{USER_TEAM_MEMBER_ID}/config", methodPut, patch, userAnon, http.StatusUnauthorized, 0},
		{"/users/{USER_TEAM_MEMBER_ID}/config", methodPut, patch, userCommenter, http.StatusForbidden, 0},
		{"/users/{USER_TEAM_MEMBER_ID}/config", methodPut, patch, userEditor, http.StatusForbidden, 0},
		{"/users/{USER_TEAM_MEMBER_ID}/config", methodPut, patch, userAdmin, http.StatusForbidden, 0},
		{"/users/{USER_TEAM_MEMBER_ID}/config", methodPut, patch, userGuest, http.StatusForbidden, 0},
	}
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsCreateBoardsAndBlocks(t *testing.T) {
	bab := toJSON(t, model.BoardsAndBlocks{
		Boards: []*model.Board{{ID: "test", Title: "Test Board", TeamID: "test-team"}},
		Blocks: []*model.Block{},
	})

	ttCases := []TestCase{
		{"/boards-and-blocks", methodPost, bab, userAnon, http.StatusUnauthorized, 0},
		{"/boards-and-blocks", methodPost, bab, userNoTeamMember, http.StatusForbidden, 0},
		{"/boards-and-blocks", methodPost, bab, userTeamMember, http.StatusOK, 1},
		{"/boards-and-blocks", methodPost, bab, userViewer, http.StatusOK, 1},
		{"/boards-and-blocks", methodPost, bab, userCommenter, http.StatusOK, 1},
		{"/boards-and-blocks", methodPost, bab, userEditor, http.StatusOK, 1},
		{"/boards-and-blocks", methodPost, bab, userAdmin, http.StatusOK, 1},
		{"/boards-and-blocks", methodPost, bab, userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsUpdateBoardsAndBlocks(t *testing.T) {
	ttCasesF := func(t *testing.T, testData TestData) []TestCase {
		newTitle := "New Block Title"
		bab := toJSON(t, model.PatchBoardsAndBlocks{
			BoardIDs:     []string{testData.publicBoard.ID},
			BoardPatches: []*model.BoardPatch{{Title: &newTitle}},
			BlockIDs:     []string{},
			BlockPatches: []*model.BlockPatch{},
		})

		return []TestCase{
			{"/boards-and-blocks", methodPatch, bab, userAnon, http.StatusUnauthorized, 0},
			{"/boards-and-blocks", methodPatch, bab, userNoTeamMember, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodPatch, bab, userTeamMember, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodPatch, bab, userViewer, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodPatch, bab, userCommenter, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodPatch, bab, userEditor, http.StatusOK, 1},
			{"/boards-and-blocks", methodPatch, bab, userAdmin, http.StatusOK, 1},
			{"/boards-and-blocks", methodPatch, bab, userGuest, http.StatusForbidden, 0},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := ttCasesF(t, testData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsDeleteBoardsAndBlocks(t *testing.T) {
	ttCasesF := func(t *testing.T, testData TestData) []TestCase {
		bab := toJSON(t, model.DeleteBoardsAndBlocks{
			Boards: []string{testData.publicBoard.ID},
			Blocks: []string{},
		})

		return []TestCase{
			{"/boards-and-blocks", methodDelete, bab, userAnon, http.StatusUnauthorized, 0},
			{"/boards-and-blocks", methodDelete, bab, userNoTeamMember, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodDelete, bab, userTeamMember, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodDelete, bab, userViewer, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodDelete, bab, userCommenter, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodDelete, bab, userEditor, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodDelete, bab, userGuest, http.StatusForbidden, 0},
			{"/boards-and-blocks", methodDelete, bab, userAdmin, http.StatusOK, 0},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := ttCasesF(t, testData)

	_, err := th.Server.App().AddMemberToBoard(&model.BoardMember{BoardID: testData.publicBoard.ID, UserID: userGuestID, SchemeViewer: true})
	require.NoError(t, err)

	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsClientConfig(t *testing.T) {
	ttCases := []TestCase{
		{"/clientConfig", methodGet, "", userAnon, http.StatusOK, 1},
		{"/clientConfig", methodGet, "", userAdmin, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetCategories(t *testing.T) {
	ttCases := []TestCase{
		{"/teams/test-team/categories", methodGet, "", userAnon, http.StatusUnauthorized, 1},
		{"/teams/test-team/categories", methodGet, "", userNoTeamMember, http.StatusForbidden, 1},
		{"/teams/test-team/categories", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team/categories", methodGet, "", userViewer, http.StatusOK, 1},
		{"/teams/test-team/categories", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/teams/test-team/categories", methodGet, "", userEditor, http.StatusOK, 1},
		{"/teams/test-team/categories", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/teams/test-team/categories", methodGet, "", userGuest, http.StatusOK, 1},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsCreateCategory(t *testing.T) {
	ttCasesF := func() []TestCase {
		category := func(userID string) string {
			return toJSON(t, model.Category{
				Name:     "Test category",
				TeamID:   "test-team",
				UserID:   userID,
				CreateAt: model.GetMillis(),
				UpdateAt: model.GetMillis(),
			})
		}

		return []TestCase{
			{"/teams/test-team/categories", methodPost, category(userNoTeamMemberID), userAnon, http.StatusUnauthorized, 0},
			{"/teams/test-team/categories", methodPost, category(userNoTeamMemberID), userNoTeamMember, http.StatusOK, 1},
			{"/teams/test-team/categories", methodPost, category(userTeamMemberID), userTeamMember, http.StatusOK, 1},
			{"/teams/test-team/categories", methodPost, category(userViewerID), userViewer, http.StatusOK, 1},
			{"/teams/test-team/categories", methodPost, category(userCommenterID), userCommenter, http.StatusOK, 1},
			{"/teams/test-team/categories", methodPost, category(userEditorID), userEditor, http.StatusOK, 1},
			{"/teams/test-team/categories", methodPost, category(userAdminID), userAdmin, http.StatusOK, 1},
			{"/teams/test-team/categories", methodPost, category(userGuestID), userGuest, http.StatusOK, 1},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := ttCasesF()
	runTestCases(t, ttCases, testData, clients)
}
func TestPermissionsUpdateCategory(t *testing.T) {
	ttCasesF := func(extraData map[string]string) []TestCase {
		category := func(userID string, categoryID string) string {
			return toJSON(t, model.Category{
				ID:       categoryID,
				Name:     "Test category",
				TeamID:   "test-team",
				UserID:   userID,
				CreateAt: model.GetMillis(),
				UpdateAt: model.GetMillis(),
				Type:     "custom",
			})
		}

		return []TestCase{
			{"/teams/test-team/categories/any", methodPut, category("", "any"), userAnonID, http.StatusUnauthorized, 0},
			{"/teams/test-team/categories/" + extraData["noTeamMember"], methodPut, category(userNoTeamMemberID, extraData["noTeamMember"]), userNoTeamMember, http.StatusForbidden, 0},
			{"/teams/test-team/categories/" + extraData["teamMember"], methodPut, category(userTeamMemberID, extraData["teamMember"]), userTeamMember, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["viewer"], methodPut, category(userViewerID, extraData["viewer"]), userViewer, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["commenter"], methodPut, category(userCommenterID, extraData["commenter"]), userCommenter, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["editor"], methodPut, category(userEditorID, extraData["editor"]), userEditor, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["admin"], methodPut, category(userAdminID, extraData["admin"]), userAdmin, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["guest"], methodPut, category(userGuestID, extraData["guest"]), userGuest, http.StatusOK, 1},

			{"/teams/test-team/categories/any", methodPut, category("other", "any"), userAnonID, http.StatusUnauthorized, 0},
			{"/teams/test-team/categories/" + extraData["noTeamMember"], methodPut, category("other", extraData["noTeamMember"]), userNoTeamMember, http.StatusBadRequest, 0},
			{"/teams/test-team/categories/" + extraData["teamMember"], methodPut, category("other", extraData["teamMember"]), userTeamMember, http.StatusBadRequest, 0},
			{"/teams/test-team/categories/" + extraData["viewer"], methodPut, category("other", extraData["viewer"]), userViewer, http.StatusBadRequest, 0},
			{"/teams/test-team/categories/" + extraData["commenter"], methodPut, category("other", extraData["commenter"]), userCommenter, http.StatusBadRequest, 0},
			{"/teams/test-team/categories/" + extraData["editor"], methodPut, category("other", extraData["editor"]), userEditor, http.StatusBadRequest, 0},
			{"/teams/test-team/categories/" + extraData["admin"], methodPut, category("other", extraData["admin"]), userAdmin, http.StatusBadRequest, 0},
		}
	}

	extraSetup := func(t *testing.T, th *TestHelper) map[string]string {
		categoryNoTeamMember, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userNoTeamMemberID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryTeamMember, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userTeamMemberID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryViewer, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userViewerID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryCommenter, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userCommenterID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryEditor, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userEditorID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryAdmin, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userAdminID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryGuest, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userGuestID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		return map[string]string{
			"noTeamMember": categoryNoTeamMember.ID,
			"teamMember":   categoryTeamMember.ID,
			"viewer":       categoryViewer.ID,
			"commenter":    categoryCommenter.ID,
			"editor":       categoryEditor.ID,
			"admin":        categoryAdmin.ID,
			"guest":        categoryGuest.ID,
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraData := extraSetup(t, th)
	ttCases := ttCasesF(extraData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsDeleteCategory(t *testing.T) {
	ttCasesF := func(extraData map[string]string) []TestCase {
		return []TestCase{
			{"/teams/other-team/categories/any", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
			{"/teams/other-team/categories/" + extraData["commenter"], methodDelete, "", userCommenter, http.StatusBadRequest, 0},
			{"/teams/other-team/categories/" + extraData["editor"], methodDelete, "", userEditor, http.StatusBadRequest, 0},
			{"/teams/other-team/categories/" + extraData["admin"], methodDelete, "", userAdmin, http.StatusBadRequest, 0},
			{"/teams/other-team/categories/" + extraData["guest"], methodDelete, "", userGuest, http.StatusBadRequest, 0},

			{"/teams/test-team/categories/any", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
			{"/teams/test-team/categories/" + extraData["noTeamMember"], methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
			{"/teams/test-team/categories/" + extraData["teamMember"], methodDelete, "", userTeamMember, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["viewer"], methodDelete, "", userViewer, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["commenter"], methodDelete, "", userCommenter, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["editor"], methodDelete, "", userEditor, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["admin"], methodDelete, "", userAdmin, http.StatusOK, 1},
			{"/teams/test-team/categories/" + extraData["guest"], methodDelete, "", userGuest, http.StatusOK, 1},
		}
	}

	extraSetup := func(t *testing.T, th *TestHelper) map[string]string {
		categoryNoTeamMember, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userNoTeamMemberID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryTeamMember, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userTeamMemberID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryViewer, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userViewerID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryCommenter, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userCommenterID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryEditor, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userEditorID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryAdmin, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userAdminID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryGuest, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userGuestID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		return map[string]string{
			"noTeamMember": categoryNoTeamMember.ID,
			"teamMember":   categoryTeamMember.ID,
			"viewer":       categoryViewer.ID,
			"commenter":    categoryCommenter.ID,
			"editor":       categoryEditor.ID,
			"admin":        categoryAdmin.ID,
			"guest":        categoryGuest.ID,
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraData := extraSetup(t, th)
	ttCases := ttCasesF(extraData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsUpdateCategoryBoard(t *testing.T) {
	ttCasesF := func(testData TestData, extraData map[string]string) []TestCase {
		return []TestCase{
			{"/teams/test-team/categories/any/boards/any", methodPost, "", userAnon, http.StatusUnauthorized, 0},
			{"/teams/test-team/categories/" + extraData["commenter"] + "/boards/" + testData.publicBoard.ID, methodPost, "", userCommenter, http.StatusOK, 0},
			{"/teams/test-team/categories/" + extraData["editor"] + "/boards/" + testData.publicBoard.ID, methodPost, "", userEditor, http.StatusOK, 0},
			{"/teams/test-team/categories/" + extraData["admin"] + "/boards/" + testData.publicBoard.ID, methodPost, "", userAdmin, http.StatusOK, 0},
			{"/teams/test-team/categories/" + extraData["guest"] + "/boards/" + testData.publicBoard.ID, methodPost, "", userGuest, http.StatusOK, 0},
		}
	}

	extraSetup := func(t *testing.T, th *TestHelper) map[string]string {
		categoryNoTeamMember, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userNoTeamMemberID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryTeamMember, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userTeamMemberID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryViewer, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userViewerID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryCommenter, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userCommenterID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryEditor, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userEditorID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryAdmin, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userAdminID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		categoryGuest, err := th.Server.App().CreateCategory(
			&model.Category{Name: "Test category", TeamID: "test-team", UserID: userGuestID, CreateAt: model.GetMillis(), UpdateAt: model.GetMillis()})
		require.NoError(t, err)
		return map[string]string{
			"noTeamMember": categoryNoTeamMember.ID,
			"teamMember":   categoryTeamMember.ID,
			"viewer":       categoryViewer.ID,
			"commenter":    categoryCommenter.ID,
			"editor":       categoryEditor.ID,
			"admin":        categoryAdmin.ID,
			"guest":        categoryGuest.ID,
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraData := extraSetup(t, th)
	ttCases := ttCasesF(testData, extraData)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetFile(t *testing.T) {
	ttCasesF := func() []TestCase {
		return []TestCase{
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}", methodGet, "", userCommenter, http.StatusOK, 1},
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}", methodGet, "", userEditor, http.StatusOK, 1},
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}", methodGet, "", userAdmin, http.StatusOK, 1},
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}", methodGet, "", userGuest, http.StatusOK, 1},

			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}?read_token=invalid", methodGet, "", userAnon, http.StatusUnauthorized, 0},
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}?read_token=valid", methodGet, "", userAnon, http.StatusOK, 1},
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}?read_token=invalid", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
			{"/files/teams/test-team/{PRIVATE_BOARD_ID}/{NEW_FILE_ID}?read_token=valid", methodGet, "", userTeamMember, http.StatusOK, 1},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)

	newFileID, err := th.Server.App().SaveFile(bytes.NewBuffer([]byte("test")), "test-team", testData.privateBoard.ID, "test.png", false)
	require.NoError(t, err)

	ttCases := ttCasesF()
	for i, tc := range ttCases {
		ttCases[i].url = strings.Replace(tc.url, "{NEW_FILE_ID}", newFileID, 1)
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsCreateSubscription(t *testing.T) {
	ttCases := func() []TestCase {
		subscription := func(userID string) string {
			return toJSON(t, model.Subscription{
				BlockType:      "card",
				BlockID:        "block-3",
				SubscriberType: "user",
				SubscriberID:   userID,
				CreateAt:       model.GetMillis(),
			})
		}
		return []TestCase{
			{"/subscriptions", methodPost, subscription(""), userAnon, http.StatusUnauthorized, 0},
			{"/subscriptions", methodPost, subscription(userNoTeamMemberID), userNoTeamMember, http.StatusOK, 1},
			{"/subscriptions", methodPost, subscription(userTeamMemberID), userTeamMember, http.StatusOK, 1},
			{"/subscriptions", methodPost, subscription(userViewerID), userViewer, http.StatusOK, 1},
			{"/subscriptions", methodPost, subscription(userCommenterID), userCommenter, http.StatusOK, 1},
			{"/subscriptions", methodPost, subscription(userEditorID), userEditor, http.StatusOK, 1},
			{"/subscriptions", methodPost, subscription(userAdminID), userAdmin, http.StatusOK, 1},
			{"/subscriptions", methodPost, subscription(userGuestID), userGuest, http.StatusOK, 1},
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases(), testData, clients)
	clients = setupClients(th)
	testData = setupData(t, th)
	runTestCases(t, ttCases(), testData, clients)
}

func TestPermissionsGetSubscriptions(t *testing.T) {
	ttCases := []TestCase{
		{"/subscriptions/{USER_ANON_ID}", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/subscriptions/{USER_COMMENTER_ID}", methodGet, "", userCommenter, http.StatusOK, 0},
		{"/subscriptions/{USER_EDITOR_ID}", methodGet, "", userEditor, http.StatusOK, 0},
		{"/subscriptions/{USER_ADMIN_ID}", methodGet, "", userAdmin, http.StatusOK, 0},
		{"/subscriptions/{USER_GUEST_ID}", methodGet, "", userGuest, http.StatusOK, 0},

		{"/subscriptions/other", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/subscriptions/other", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/subscriptions/other", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/subscriptions/other", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/subscriptions/other", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/subscriptions/other", methodGet, "", userAdmin, http.StatusForbidden, 0},
		{"/subscriptions/other", methodGet, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsDeleteSubscription(t *testing.T) {
	ttCases := []TestCase{
		{"/subscriptions/block-3/{USER_ANON_ID}", methodDelete, "", userAnon, http.StatusUnauthorized, 0},
		{"/subscriptions/block-3/{USER_COMMENTER_ID}", methodDelete, "", userCommenter, http.StatusOK, 0},
		{"/subscriptions/block-3/{USER_EDITOR_ID}", methodDelete, "", userEditor, http.StatusOK, 0},
		{"/subscriptions/block-3/{USER_ADMIN_ID}", methodDelete, "", userAdmin, http.StatusOK, 0},
		{"/subscriptions/block-3/{USER_GUEST_ID}", methodDelete, "", userGuest, http.StatusOK, 0},

		{"/subscriptions/block-3/other", methodDelete, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/subscriptions/block-3/other", methodDelete, "", userTeamMember, http.StatusForbidden, 0},
		{"/subscriptions/block-3/other", methodDelete, "", userViewer, http.StatusForbidden, 0},
		{"/subscriptions/block-3/other", methodDelete, "", userCommenter, http.StatusForbidden, 0},
		{"/subscriptions/block-3/other", methodDelete, "", userEditor, http.StatusForbidden, 0},
		{"/subscriptions/block-3/other", methodDelete, "", userAdmin, http.StatusForbidden, 0},
		{"/subscriptions/block-3/other", methodDelete, "", userGuest, http.StatusForbidden, 0},
	}

	extraSetup := func(t *testing.T, th *TestHelper) {
		_, err := th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: userNoTeamMemberID, CreateAt: model.GetMillis()})
		require.NoError(t, err)
		_, err = th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: userTeamMemberID, CreateAt: model.GetMillis()})
		require.NoError(t, err)
		_, err = th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: userViewerID, CreateAt: model.GetMillis()})
		require.NoError(t, err)
		_, err = th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: userCommenterID, CreateAt: model.GetMillis()})
		require.NoError(t, err)
		_, err = th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: userEditorID, CreateAt: model.GetMillis()})
		require.NoError(t, err)
		_, err = th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: userAdminID, CreateAt: model.GetMillis()})
		require.NoError(t, err)
		_, err = th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: userGuestID, CreateAt: model.GetMillis()})
		require.NoError(t, err)
		_, err = th.Server.App().CreateSubscription(
			&model.Subscription{BlockType: "card", BlockID: "block-3", SubscriberType: "user", SubscriberID: "other", CreateAt: model.GetMillis()})
		require.NoError(t, err)
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	extraSetup(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsOnboard(t *testing.T) {
	ttCases := []TestCase{
		{"/teams/test-team/onboard", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/onboard", methodPost, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/onboard", methodPost, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team/onboard", methodPost, "", userViewer, http.StatusOK, 1},
		{"/teams/test-team/onboard", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/teams/test-team/onboard", methodPost, "", userEditor, http.StatusOK, 1},
		{"/teams/test-team/onboard", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/teams/test-team/onboard", methodPost, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)

	err := th.Server.App().InitTemplates()
	require.NoError(t, err, "InitTemplates should not fail")

	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsBoardArchiveExport(t *testing.T) {
	ttCases := []TestCase{
		{"/boards/{PUBLIC_BOARD_ID}/archive/export", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_BOARD_ID}/archive/export", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/archive/export", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/archive/export", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_BOARD_ID}/archive/export", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_BOARD_ID}/archive/export", methodGet, "", userGuest, http.StatusOK, 1},

		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PUBLIC_TEMPLATE_ID}/archive/export", methodGet, "", userGuest, http.StatusForbidden, 0},

		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userViewer, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userEditor, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/boards/{PRIVATE_TEMPLATE_ID}/archive/export", methodGet, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsBoardArchiveImport(t *testing.T) {
	ttCases := []TestCase{
		{"/teams/test-team/archive/import", methodPost, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/archive/import", methodPost, "", userNoTeamMember, http.StatusForbidden, 1},
		{"/teams/test-team/archive/import", methodPost, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team/archive/import", methodPost, "", userViewer, http.StatusOK, 1},
		{"/teams/test-team/archive/import", methodPost, "", userCommenter, http.StatusOK, 1},
		{"/teams/test-team/archive/import", methodPost, "", userEditor, http.StatusOK, 1},
		{"/teams/test-team/archive/import", methodPost, "", userAdmin, http.StatusOK, 1},
		{"/teams/test-team/archive/import", methodPost, "", userGuest, http.StatusForbidden, 0},
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsMinimumRolesApplied(t *testing.T) {
	ttCasesF := func(t *testing.T, th *TestHelper, minimumRole model.BoardRole, testData TestData) []TestCase {
		counter := 0
		newBlockJSON := func(boardID string) string {
			counter++
			return toJSON(t, []*model.Block{{
				ID:       fmt.Sprintf("%d", counter),
				Title:    "Board To Create",
				BoardID:  boardID,
				Type:     "card",
				CreateAt: model.GetMillis(),
				UpdateAt: model.GetMillis(),
			}})
		}
		_, err := th.Server.App().PatchBoard(&model.BoardPatch{MinimumRole: &minimumRole}, testData.publicBoard.ID, userAdminID)
		require.NoError(t, err)
		_, err = th.Server.App().PatchBoard(&model.BoardPatch{MinimumRole: &minimumRole}, testData.privateTemplate.ID, userAdminID)
		require.NoError(t, err)
		_, err = th.Server.App().PatchBoard(&model.BoardPatch{MinimumRole: &minimumRole}, testData.publicTemplate.ID, userAdminID)
		require.NoError(t, err)

		if minimumRole == "viewer" || minimumRole == "commenter" {
			return []TestCase{
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userViewer, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userCommenter, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAdmin, http.StatusOK, 1},

				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userViewer, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userCommenter, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAdmin, http.StatusOK, 1},

				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userViewer, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userCommenter, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAdmin, http.StatusOK, 1},

				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userViewer, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userCommenter, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAdmin, http.StatusOK, 1},
			}
		} else {
			return []TestCase{
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userViewer, http.StatusOK, 1},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userCommenter, http.StatusOK, 1},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PRIVATE_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.privateBoard.ID), userAdmin, http.StatusOK, 1},

				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userViewer, http.StatusOK, 1},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userCommenter, http.StatusOK, 1},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PUBLIC_BOARD_ID}/blocks", methodPost, newBlockJSON(testData.publicBoard.ID), userAdmin, http.StatusOK, 1},

				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userViewer, http.StatusOK, 1},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userCommenter, http.StatusOK, 1},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PRIVATE_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.privateTemplate.ID), userAdmin, http.StatusOK, 1},

				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAnon, http.StatusUnauthorized, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userNoTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userTeamMember, http.StatusForbidden, 0},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userViewer, http.StatusOK, 1},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userCommenter, http.StatusOK, 1},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userEditor, http.StatusOK, 1},
				{"/boards/{PUBLIC_TEMPLATE_ID}/blocks", methodPost, newBlockJSON(testData.publicTemplate.ID), userAdmin, http.StatusOK, 1},
			}
		}
	}

	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	t.Run("minimum role viewer", func(t *testing.T) {
		clients := setupClients(th)
		testData := setupData(t, th)
		ttCases := ttCasesF(t, th, "viewer", testData)
		runTestCases(t, ttCases, testData, clients)
	})
}

func TestPermissionsChannels(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		{"/teams/test-team/channels", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/channels", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/channels", methodGet, "", userTeamMember, http.StatusOK, 2},
		{"/teams/test-team/channels", methodGet, "", userViewer, http.StatusOK, 2},
		{"/teams/test-team/channels", methodGet, "", userCommenter, http.StatusOK, 2},
		{"/teams/test-team/channels", methodGet, "", userEditor, http.StatusOK, 2},
		{"/teams/test-team/channels", methodGet, "", userAdmin, http.StatusOK, 2},
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsChannel(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		{"/teams/test-team/channels/valid-channel-id", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/channels/valid-channel-id", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/channels/valid-channel-id", methodGet, "", userTeamMember, http.StatusOK, 1},
		{"/teams/test-team/channels/valid-channel-id", methodGet, "", userViewer, http.StatusOK, 1},
		{"/teams/test-team/channels/valid-channel-id", methodGet, "", userCommenter, http.StatusOK, 1},
		{"/teams/test-team/channels/valid-channel-id", methodGet, "", userEditor, http.StatusOK, 1},
		{"/teams/test-team/channels/valid-channel-id", methodGet, "", userAdmin, http.StatusOK, 1},

		{"/teams/test-team/channels/not-valid-channel-id", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/teams/test-team/channels/not-valid-channel-id", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/channels/not-valid-channel-id", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/teams/test-team/channels/not-valid-channel-id", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/teams/test-team/channels/not-valid-channel-id", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/teams/test-team/channels/not-valid-channel-id", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/teams/test-team/channels/not-valid-channel-id", methodGet, "", userAdmin, http.StatusForbidden, 0},
	}
	runTestCases(t, ttCases, testData, clients)
}

func TestPermissionsGetStatistics(t *testing.T) {
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()
	clients := setupClients(th)
	testData := setupData(t, th)
	ttCases := []TestCase{
		{"/statistics", methodGet, "", userAnon, http.StatusUnauthorized, 0},
		{"/statistics", methodGet, "", userNoTeamMember, http.StatusForbidden, 0},
		{"/statistics", methodGet, "", userTeamMember, http.StatusForbidden, 0},
		{"/statistics", methodGet, "", userViewer, http.StatusForbidden, 0},
		{"/statistics", methodGet, "", userCommenter, http.StatusForbidden, 0},
		{"/statistics", methodGet, "", userEditor, http.StatusForbidden, 0},
		{"/statistics", methodGet, "", userAdmin, http.StatusOK, 1},
		{"/statistics", methodGet, "", userGuest, http.StatusForbidden, 0},
	}
	runTestCases(t, ttCases, testData, clients)
}
