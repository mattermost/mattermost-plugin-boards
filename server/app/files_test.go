// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	mm_model "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore/mocks"
)

const (
	testFileName = "temp-file-name"
	testPath     = "/path/to/file/fileName.txt"
)

var testBoardID = utils.NewID(utils.IDTypeBoard)

var errDummy = errors.New("hello")

type TestError struct{}

func (err *TestError) Error() string { return "Mocked File backend error" }

func TestGetFileReader(t *testing.T) {
	validTeamID := "abcdefghijklmnopqrstuvwxyz" // 26 chars - valid Mattermost ID
	testFilePath := filepath.Join(validTeamID, testBoardID, testFileName)

	th, _ := SetupTestHelper(t)
	mockedReadCloseSeek := &mocks.ReadCloseSeeker{}
	t.Run("should get file reader from filestore successfully", func(t *testing.T) {
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		readerFunc := func(path string) filestore.ReadCloseSeeker {
			return mockedReadCloseSeek
		}

		readerErrorFunc := func(path string) error {
			return nil
		}

		fileExistsFunc := func(path string) bool {
			return true
		}

		fileExistsErrorFunc := func(path string) error {
			return nil
		}

		mockedFileBackend.On("Reader", testFilePath).Return(readerFunc, readerErrorFunc)
		mockedFileBackend.On("FileExists", testFilePath).Return(fileExistsFunc, fileExistsErrorFunc)
		actual, _ := th.App.GetFileReader(validTeamID, testBoardID, testFileName)
		assert.Equal(t, mockedReadCloseSeek, actual)
	})

	t.Run("should get error from filestore when file exists return error", func(t *testing.T) {
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedError := &TestError{}
		readerFunc := func(path string) filestore.ReadCloseSeeker {
			return mockedReadCloseSeek
		}

		readerErrorFunc := func(path string) error {
			return nil
		}

		fileExistsFunc := func(path string) bool {
			return false
		}

		fileExistsErrorFunc := func(path string) error {
			return mockedError
		}

		mockedFileBackend.On("Reader", testFilePath).Return(readerFunc, readerErrorFunc)
		mockedFileBackend.On("FileExists", testFilePath).Return(fileExistsFunc, fileExistsErrorFunc)
		actual, err := th.App.GetFileReader(validTeamID, testBoardID, testFileName)
		assert.Error(t, err, mockedError)
		assert.Nil(t, actual)
	})

	t.Run("should return error, if get reader from file backend returns error", func(t *testing.T) {
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedError := &TestError{}
		readerFunc := func(path string) filestore.ReadCloseSeeker {
			return nil
		}

		readerErrorFunc := func(path string) error {
			return mockedError
		}

		fileExistsFunc := func(path string) bool {
			return false
		}

		fileExistsErrorFunc := func(path string) error {
			return nil
		}

		mockedFileBackend.On("Reader", testFilePath).Return(readerFunc, readerErrorFunc)
		mockedFileBackend.On("FileExists", testFilePath).Return(fileExistsFunc, fileExistsErrorFunc)
		actual, err := th.App.GetFileReader(validTeamID, testBoardID, testFileName)
		assert.Error(t, err, mockedError)
		assert.Nil(t, actual)
	})

	t.Run("should move file from old filepath to new filepath, if file doesnot exists in new filepath and workspace id is 0", func(t *testing.T) {
		filePath := filepath.Join("0", testBoardID, testFileName)
		workspaceid := "0"
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		readerFunc := func(path string) filestore.ReadCloseSeeker {
			return mockedReadCloseSeek
		}

		readerErrorFunc := func(path string) error {
			return nil
		}

		fileExistsFunc := func(path string) bool {
			// return true for old path
			return path == testFileName
		}

		fileExistsErrorFunc := func(path string) error {
			return nil
		}

		moveFileFunc := func(oldFileName, newFileName string) error {
			return nil
		}

		// Add mock for GetBoard call since workspaceid is "0" (model.GlobalTeamID)
		th.Store.EXPECT().GetBoard(testBoardID).Return(&model.Board{
			ID:         testBoardID,
			IsTemplate: true, // Set to true since it's using GlobalTeamID
		}, nil)

		mockedFileBackend.On("FileExists", filePath).Return(fileExistsFunc, fileExistsErrorFunc)
		mockedFileBackend.On("FileExists", testFileName).Return(fileExistsFunc, fileExistsErrorFunc)
		mockedFileBackend.On("MoveFile", testFileName, filePath).Return(moveFileFunc)
		mockedFileBackend.On("Reader", filePath).Return(readerFunc, readerErrorFunc)

		actual, _ := th.App.GetFileReader(workspaceid, testBoardID, testFileName)
		assert.Equal(t, mockedReadCloseSeek, actual)
	})

	t.Run("should return file reader, if file doesnot exists in new filepath and old file path", func(t *testing.T) {
		filePath := filepath.Join("0", testBoardID, testFileName)
		fileName := testFileName
		workspaceid := "0"
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		readerFunc := func(path string) filestore.ReadCloseSeeker {
			return mockedReadCloseSeek
		}

		readerErrorFunc := func(path string) error {
			return nil
		}

		fileExistsFunc := func(path string) bool {
			// return true for old path
			return false
		}

		fileExistsErrorFunc := func(path string) error {
			return nil
		}

		moveFileFunc := func(oldFileName, newFileName string) error {
			return nil
		}

		// Add mock for GetBoard call since workspaceid is "0" (model.GlobalTeamID)
		th.Store.EXPECT().GetBoard(testBoardID).Return(&model.Board{
			ID:         testBoardID,
			IsTemplate: true, // Set to true since it's using GlobalTeamID
		}, nil)

		mockedFileBackend.On("FileExists", filePath).Return(fileExistsFunc, fileExistsErrorFunc)
		mockedFileBackend.On("FileExists", testFileName).Return(fileExistsFunc, fileExistsErrorFunc)
		mockedFileBackend.On("MoveFile", fileName, filePath).Return(moveFileFunc)
		mockedFileBackend.On("Reader", filePath).Return(readerFunc, readerErrorFunc)

		actual, _ := th.App.GetFileReader(workspaceid, testBoardID, testFileName)
		assert.Equal(t, mockedReadCloseSeek, actual)
	})
}

func TestSaveFile(t *testing.T) {
	th, _ := SetupTestHelper(t)
	mockedReadCloseSeek := &mocks.ReadCloseSeeker{}

	t.Run("should save file to file store with boardID in path", func(t *testing.T) {
		fileName := "temp-file-name.txt"
		validTeamID := mm_model.NewId()
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend

		th.Store.EXPECT().SaveFileInfo(gomock.Any()).DoAndReturn(func(info *mm_model.FileInfo) error {
			parts := strings.Split(filepath.ToSlash(info.Path), "/")
			assert.Equal(t, "boards", parts[0])
			assert.Equal(t, testBoardID, parts[2]) // boardID is the 3rd path component
			fileName = parts[len(parts)-1]
			return nil
		})

		writeFileFunc := func(reader io.Reader, path string) int64 { return int64(10) }
		writeFileErrorFunc := func(reader io.Reader, filePath string) error { return nil }
		mockedFileBackend.On("WriteFile", mockedReadCloseSeek, mock.Anything).Return(writeFileFunc, writeFileErrorFunc)
		actual, err := th.App.SaveFile(mockedReadCloseSeek, validTeamID, testBoardID, fileName, false)
		assert.Equal(t, fileName, actual)
		assert.Nil(t, err)
	})

	t.Run("should save .jpeg file as .jpg with boardID in path", func(t *testing.T) {
		fileName := "temp-file-name.jpeg"
		validTeamID := mm_model.NewId()
		validBoardID := utils.NewID(utils.IDTypeBoard)
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend

		th.Store.EXPECT().SaveFileInfo(gomock.Any()).DoAndReturn(func(info *mm_model.FileInfo) error {
			normalizedPath := filepath.ToSlash(info.Path)
			assert.Contains(t, normalizedPath, validBoardID)
			assert.Equal(t, "jpg", strings.Split(normalizedPath, ".")[1])
			return nil
		})

		writeFileFunc := func(reader io.Reader, path string) int64 { return int64(10) }
		writeFileErrorFunc := func(reader io.Reader, filePath string) error { return nil }
		mockedFileBackend.On("WriteFile", mockedReadCloseSeek, mock.Anything).Return(writeFileFunc, writeFileErrorFunc)
		actual, err := th.App.SaveFile(mockedReadCloseSeek, validTeamID, validBoardID, fileName, false)
		assert.Nil(t, err)
		assert.NotNil(t, actual)
	})

	t.Run("should return error when fileBackend.WriteFile returns error", func(t *testing.T) {
		fileName := "temp-file-name.jpeg"
		validTeamID := mm_model.NewId()
		validBoardID := utils.NewID(utils.IDTypeBoard)
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedError := &TestError{}

		writeFileFunc := func(reader io.Reader, path string) int64 {
			return int64(10)
		}

		writeFileErrorFunc := func(reader io.Reader, filePath string) error {
			return mockedError
		}

		mockedFileBackend.On("WriteFile", mockedReadCloseSeek, mock.Anything).Return(writeFileFunc, writeFileErrorFunc)
		actual, err := th.App.SaveFile(mockedReadCloseSeek, validTeamID, validBoardID, fileName, false)
		assert.Equal(t, "", actual)
		assert.Equal(t, "unable to store the file in the files storage: Mocked File backend error", err.Error())
	})
}

func TestGetFileInfo(t *testing.T) {
	th, _ := SetupTestHelper(t)

	t.Run("should return file info", func(t *testing.T) {
		fileInfo := &mm_model.FileInfo{
			Id:       "file_info_id",
			Archived: false,
		}

		th.Store.EXPECT().GetFileInfo("filename").Return(fileInfo, nil).Times(2)

		fetchedFileInfo, err := th.App.GetFileInfo("Afilename")
		assert.NoError(t, err)
		assert.Equal(t, "file_info_id", fetchedFileInfo.Id)
		assert.False(t, fetchedFileInfo.Archived)

		fetchedFileInfo, err = th.App.GetFileInfo("Afilename.txt")
		assert.NoError(t, err)
		assert.Equal(t, "file_info_id", fetchedFileInfo.Id)
		assert.False(t, fetchedFileInfo.Archived)
	})

	t.Run("should return archived file info", func(t *testing.T) {
		fileInfo := &mm_model.FileInfo{
			Id:       "file_info_id",
			Archived: true,
		}

		th.Store.EXPECT().GetFileInfo("filename").Return(fileInfo, nil)

		fetchedFileInfo, err := th.App.GetFileInfo("Afilename")
		assert.NoError(t, err)
		assert.Equal(t, "file_info_id", fetchedFileInfo.Id)
		assert.True(t, fetchedFileInfo.Archived)
	})

	t.Run("should return archived file infoerror", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("filename").Return(nil, errDummy)

		fetchedFileInfo, err := th.App.GetFileInfo("Afilename")
		assert.Error(t, err)
		assert.Nil(t, fetchedFileInfo)
	})
}

func TestGetFile(t *testing.T) {
	th, _ := SetupTestHelper(t)
	validTeamID := "abcdefghijklmnopqrstuvwxyz" // 26 chars (valid Mattermost ID)

	t.Run("happy path, no errors", func(t *testing.T) {
		validBoardID := utils.NewID(utils.IDTypeBoard)
		fileName := "7fileInfoID.txt"
		expectedPath := filepath.Join(validTeamID, validBoardID, fileName)
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: expectedPath,
		}, nil).Times(2)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedReadCloseSeek := &mocks.ReadCloseSeeker{}
		readerFunc := func(path string) filestore.ReadCloseSeeker {
			return mockedReadCloseSeek
		}

		readerErrorFunc := func(path string) error {
			return nil
		}
		mockedFileBackend.On("Reader", expectedPath).Return(readerFunc, readerErrorFunc)
		mockedFileBackend.On("FileExists", expectedPath).Return(true, nil)

		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, fileName)
		assert.NoError(t, err)
		assert.NotNil(t, fileInfo)
		assert.NotNil(t, seeker)
	})

	t.Run("when GetFilePath() throws error", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(nil, errDummy)

		validBoardID := utils.NewID(utils.IDTypeBoard)
		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.Error(t, err)
		assert.Nil(t, fileInfo)
		assert.Nil(t, seeker)
	})

	t.Run("when FileExists returns false", func(t *testing.T) {
		validBoardID := utils.NewID(utils.IDTypeBoard)
		fileName := "7fileInfoID.txt"
		expectedPath := filepath.Join(validTeamID, validBoardID, fileName)
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: expectedPath,
		}, nil).Times(2)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", expectedPath).Return(false, nil)

		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, fileName)
		assert.Error(t, err)
		assert.Nil(t, fileInfo)
		assert.Nil(t, seeker)
	})
	t.Run("when FileReader throws error", func(t *testing.T) {
		validBoardID := utils.NewID(utils.IDTypeBoard)
		fileName := "7fileInfoID.txt"
		expectedPath := filepath.Join(validTeamID, validBoardID, fileName)
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: expectedPath,
		}, nil).Times(2)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("Reader", expectedPath).Return(nil, errDummy)
		mockedFileBackend.On("FileExists", expectedPath).Return(true, nil)

		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, fileName)
		assert.Error(t, err)
		assert.Nil(t, fileInfo)
		assert.Nil(t, seeker)
	})
}

func TestGetFilePath(t *testing.T) {
	th, _ := SetupTestHelper(t)

	t.Run("when FileInfo exists", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: testPath,
		}, nil)

		validTeamID := "abcdefghijklmnopqrstuvwxyz" // 26 chars - valid Mattermost ID
		validBoardID := utils.NewID(utils.IDTypeBoard)
		fileInfo, filePath, err := th.App.GetFilePath(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.NoError(t, err)
		assert.NotNil(t, fileInfo)
		assert.Equal(t, testPath, filePath)
	})

	t.Run("when FileInfo doesn't exist", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(nil, nil)

		validTeamID := "abcdefghijklmnopqrstuvwxyz" // 26 chars - valid Mattermost ID
		validBoardID := utils.NewID(utils.IDTypeBoard)
		fileInfo, filePath, err := th.App.GetFilePath(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.NoError(t, err)
		assert.Nil(t, fileInfo)
		assert.Equal(t, validTeamID+"/"+validBoardID+"/7fileInfoID.txt", filePath)
	})

	t.Run("when FileInfo exists but FileInfo.Path is not set", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: "",
		}, nil)

		validTeamID := "abcdefghijklmnopqrstuvwxyz" // 26 chars - valid Mattermost ID
		validBoardID := utils.NewID(utils.IDTypeBoard)
		fileInfo, filePath, err := th.App.GetFilePath(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.NoError(t, err)
		assert.NotNil(t, fileInfo)
		assert.Equal(t, validTeamID+"/"+validBoardID+"/7fileInfoID.txt", filePath)
	})
}

func TestCopyCard(t *testing.T) {
	th, _ := SetupTestHelper(t)
	imageBlock := &model.Block{
		ID:         "imageBlock",
		ParentID:   "c3zqnh6fsu3f4mr6hzq9hizwske",
		CreatedBy:  "6k6ynxdp47dujjhhojw9nqhmyh",
		ModifiedBy: "6k6ynxdp47dujjhhojw9nqhmyh",
		Schema:     1,
		Type:       "image",
		Title:      "",
		Fields:     map[string]interface{}{"fileId": "7fileName123456789012345678.jpg"},
		CreateAt:   1680725585250,
		UpdateAt:   1680725585250,
		DeleteAt:   0,
		BoardID:    "bvalidtestboard123456789012",
	}
	validTestBoardID := "bvalidtestboard123456789012" // 27-char valid board ID

	t.Run("Board doesn't exist", func(t *testing.T) {
		th.Store.EXPECT().GetBoard(validTestBoardID).Return(nil, errDummy)
		_, err := th.App.CopyCardFiles(validTestBoardID, []*model.Block{}, false)
		assert.Error(t, err)
	})

	t.Run("Board exists, image block, with FileInfo", func(t *testing.T) {
		fileInfo := &mm_model.FileInfo{
			Id:   "imageBlock",
			Path: testPath,
		}
		th.Store.EXPECT().GetBoard(validTestBoardID).Return(&model.Board{
			ID:         validTestBoardID,
			TeamID:     "validteam12345678901234567",
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo("fileName123456789012345678").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", mock.Anything).Return(true, nil).Twice()
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		updatedFileNames, err := th.App.CopyCardFiles(validTestBoardID, []*model.Block{imageBlock}, false)
		assert.NoError(t, err)
		assert.Equal(t, "7fileName123456789012345678.jpg", imageBlock.Fields["fileId"])
		assert.NotNil(t, updatedFileNames["7fileName123456789012345678.jpg"])
		assert.NotNil(t, updatedFileNames[imageBlock.Fields["fileId"].(string)])
	})

	t.Run("Board exists, attachment block, with FileInfo", func(t *testing.T) {
		attachmentBlock := &model.Block{
			ID:         "attachmentBlock",
			ParentID:   "c3zqnh6fsu3f4mr6hzq9hizwske",
			CreatedBy:  "6k6ynxdp47dujjhhojw9nqhmyh",
			ModifiedBy: "6k6ynxdp47dujjhhojw9nqhmyh",
			Schema:     1,
			Type:       "attachment",
			Title:      "",
			Fields:     map[string]interface{}{"fileId": "7fileName123456789012345678.jpg"},
			CreateAt:   1680725585250,
			UpdateAt:   1680725585250,
			DeleteAt:   0,
			BoardID:    validTestBoardID,
		}

		fileInfo := &mm_model.FileInfo{
			Id:   "attachmentBlock",
			Path: testPath,
		}
		th.Store.EXPECT().GetBoard(validTestBoardID).Return(&model.Board{
			ID:         validTestBoardID,
			TeamID:     "validteam12345678901234567",
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo("fileName123456789012345678").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", mock.Anything).Return(true, nil).Twice()
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		updatedFileNames, err := th.App.CopyCardFiles(validTestBoardID, []*model.Block{attachmentBlock}, false)
		assert.NoError(t, err)
		assert.NotNil(t, updatedFileNames[imageBlock.Fields["fileId"].(string)])
	})

	t.Run("Board exists, image block, without FileInfo", func(t *testing.T) {
		boardID := imageBlock.BoardID
		th.Store.EXPECT().GetBoard(boardID).Return(&model.Board{
			ID:         boardID,
			TeamID:     "validteam12345678901234567",
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo(gomock.Any()).Return(nil, nil)
		th.Store.EXPECT().SaveFileInfo(gomock.Any()).Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", mock.Anything).Return(true, nil).Twice()
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		updatedFileNames, err := th.App.CopyCardFiles(boardID, []*model.Block{imageBlock}, false)
		assert.NoError(t, err)
		assert.NotNil(t, imageBlock.Fields["fileId"].(string))
		assert.NotNil(t, updatedFileNames[imageBlock.Fields["fileId"].(string)])
	})
}

func TestCopyAndUpdateCardFiles(t *testing.T) {
	th, _ := SetupTestHelper(t)
	imageBlock := &model.Block{
		ID:         "imageBlock",
		ParentID:   "c3zqnh6fsu3f4mr6hzq9hizwske",
		CreatedBy:  "6k6ynxdp47dujjhhojw9nqhmyh",
		ModifiedBy: "6k6ynxdp47dujjhhojw9nqhmyh",
		Schema:     1,
		Type:       "image",
		Title:      "",
		Fields:     map[string]interface{}{"fileId": "7fileName123456789012345678.jpg"},
		CreateAt:   1680725585250,
		UpdateAt:   1680725585250,
		DeleteAt:   0,
		BoardID:    "bvalidtestboard123456789012",
	}

	validTestBoardID2 := "bvalidtestboard123456789012" // 27-char valid board ID

	validImageBlock := &model.Block{
		ID:         "validImageBlock",
		ParentID:   "c3zqnh6fsu3f4mr6hzq9hizwske",
		CreatedBy:  "6k6ynxdp47dujjhhojw9nqhmyh",
		ModifiedBy: "6k6ynxdp47dujjhhojw9nqhmyh",
		Schema:     1,
		Type:       "image",
		Title:      "",
		Fields:     map[string]interface{}{"fileId": "7xhwgf5r15fr3dryfozf1dmy41r.png"},
		CreateAt:   1680725585250,
		UpdateAt:   1680725585250,
		DeleteAt:   0,
		BoardID:    validTestBoardID2,
	}

	invalidShortFileIDBlock := &model.Block{
		ID:         "invalidShortFileIDBlock",
		ParentID:   "c3zqnh6fsu3f4mr6hzq9hizwske",
		CreatedBy:  "6k6ynxdp47dujjhhojw9nqhmyh",
		ModifiedBy: "6k6ynxdp47dujjhhojw9nqhmyh",
		Schema:     1,
		Type:       "image",
		Title:      "",
		Fields:     map[string]interface{}{"fileId": "7short.png"},
		CreateAt:   1680725585250,
		UpdateAt:   1680725585250,
		DeleteAt:   0,
		BoardID:    validTestBoardID2,
	}

	emptyFileBlock := &model.Block{
		ID:         "emptyFileBlock",
		ParentID:   "c3zqnh6fsu3f4mr6hzq9hizwske",
		CreatedBy:  "6k6ynxdp47dujjhhojw9nqhmyh",
		ModifiedBy: "6k6ynxdp47dujjhhojw9nqhmyh",
		Schema:     1,
		Type:       "image",
		Title:      "",
		Fields:     map[string]interface{}{"fileId": ""},
		CreateAt:   1680725585250,
		UpdateAt:   1680725585250,
		DeleteAt:   0,
		BoardID:    validTestBoardID2,
	}

	t.Run("Board exists, image block, with FileInfo", func(t *testing.T) {
		fileInfo := &mm_model.FileInfo{
			Id:   "imageBlock",
			Path: testPath,
		}
		th.Store.EXPECT().GetBoard("bvalidtestboard123456789012").Return(&model.Board{
			ID:         "bvalidtestboard123456789012",
			TeamID:     "validteam12345678901234567",
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo("fileName123456789012345678").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)
		th.Store.EXPECT().PatchBlocks(gomock.Any(), "userID").Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", mock.Anything).Return(true, nil).Twice()
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		err := th.App.CopyAndUpdateCardFiles("bvalidtestboard123456789012", "userID", []*model.Block{imageBlock}, false)
		assert.NoError(t, err)

		assert.NotEqual(t, testPath, imageBlock.Fields["fileId"])
	})

	t.Run("Valid file ID", func(t *testing.T) {
		fileInfo := &mm_model.FileInfo{
			Id:   "validImageBlock",
			Path: testPath,
		}
		th.Store.EXPECT().GetBoard(validTestBoardID2).Return(&model.Board{ID: validTestBoardID2, TeamID: "validteam12345678901234567", IsTemplate: false}, nil)
		th.Store.EXPECT().GetFileInfo("xhwgf5r15fr3dryfozf1dmy41r").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)
		th.Store.EXPECT().PatchBlocks(gomock.Any(), "userID").Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", mock.Anything).Return(true, nil).Twice()
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		err := th.App.CopyAndUpdateCardFiles(validTestBoardID2, "userID", []*model.Block{validImageBlock}, false)
		assert.NoError(t, err)
	})

	t.Run("Invalid file ID length", func(t *testing.T) {
		th.Store.EXPECT().GetBoard(validTestBoardID2).Return(&model.Board{ID: validTestBoardID2, TeamID: "validteam12345678901234567", IsTemplate: false}, nil)
		err := th.App.CopyAndUpdateCardFiles(validTestBoardID2, "userID", []*model.Block{invalidShortFileIDBlock}, false)
		assert.Error(t, err)
		assert.True(t,
			strings.Contains(err.Error(), "Invalid Block ID") ||
				strings.Contains(err.Error(), "Could not validate file ID"),
			"Expected error message to contain 'Invalid Block ID' or 'Could not validate file ID', got: %s", err.Error())
	})

	t.Run("Empty file ID", func(t *testing.T) {
		// blocks with an empty fileId/attachmentId should be treated
		// as having no attached file
		th.Store.EXPECT().GetBoard(validTestBoardID2).Return(&model.Board{ID: validTestBoardID2, TeamID: "validteam12345678901234567", IsTemplate: false}, nil)
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		err := th.App.CopyAndUpdateCardFiles(validTestBoardID2, "userID", []*model.Block{emptyFileBlock}, false)
		assert.NoError(t, err)
	})
}

func TestCopyCardFiles(t *testing.T) {
	th, _ := SetupTestHelper(t)

	t.Run("ValidFileID", func(t *testing.T) {
		sourceBoardID := utils.NewID(utils.IDTypeBoard)
		destBoardID := utils.NewID(utils.IDTypeBoard)
		validMattermostID := mm_model.NewId()           // 26-char valid Mattermost ID
		validFileID := "7" + validMattermostID + ".jpg" // Valid file ID: '7' + 26-char ID + extension
		fileInfoID := validMattermostID                 // GetFileInfo extracts ID by removing '7' prefix and extension
		copiedBlocks := []*model.Block{
			{
				Type:    model.TypeImage,
				Fields:  map[string]interface{}{"fileId": validFileID},
				BoardID: destBoardID,
			},
		}

		teamID := mm_model.NewId()
		th.Store.EXPECT().GetBoard(sourceBoardID).Return(&model.Board{
			ID:         sourceBoardID,
			TeamID:     teamID,
			IsTemplate: false,
		}, nil)
		// If block.BoardID != sourceBoardID, GetBoard is called again with block.BoardID
		th.Store.EXPECT().GetBoard(destBoardID).Return(&model.Board{
			ID:         destBoardID,
			TeamID:     teamID,
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo(fileInfoID).Return(nil, nil)
		th.Store.EXPECT().SaveFileInfo(gomock.Any()).Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", mock.Anything).Return(true, nil).Twice()
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		newFileNames, err := th.App.CopyCardFiles(sourceBoardID, copiedBlocks, false)

		assert.NoError(t, err)
		assert.NotNil(t, newFileNames)
	})

	t.Run("InvalidFileID", func(t *testing.T) {
		sourceBoardID := "bsourceboard123456789012345"
		destBoardID := "bdestinationboard1234567890"
		copiedBlocks := []*model.Block{
			{
				Type:    model.TypeImage,
				Fields:  map[string]interface{}{"fileId": "../../../../../filePath"},
				BoardID: destBoardID,
			},
		}

		th.Store.EXPECT().GetBoard(sourceBoardID).Return(&model.Board{
			ID:         sourceBoardID,
			TeamID:     "validteam12345678901234567",
			IsTemplate: false,
		}, nil)

		newFileNames, err := th.App.CopyCardFiles(sourceBoardID, copiedBlocks, false)

		assert.Error(t, err)
		assert.Nil(t, newFileNames)
	})
}

func TestGetDestinationFilePath(t *testing.T) {
	validTeamID := "abcdefghijklmnopqrstuvwxyz"
	validBoardID := "babcdefghijklmnopqrstuvwxyz" // 27-char board ID starting with 'b'

	t.Run("Should reject path traversal in template teamID", func(t *testing.T) {
		result, err := getDestinationFilePath(true, "../../../etc", validBoardID, "filename")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid teamID in ValidateTeamID")
		assert.Equal(t, "", result)
	})

	t.Run("Should reject path traversal in template boardID", func(t *testing.T) {
		result, err := getDestinationFilePath(true, validTeamID, "../../../etc", "filename")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid boardID")
		assert.Equal(t, "", result)
	})

	t.Run("Should reject path traversal in template filename", func(t *testing.T) {
		result, err := getDestinationFilePath(true, validTeamID, validBoardID, "../../../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid filename")
		assert.Equal(t, "", result)
	})

	t.Run("Should use direct path for templates (to avoid data retention)", func(t *testing.T) {
		result, err := getDestinationFilePath(true, validTeamID, validBoardID, "validFile")
		assert.NoError(t, err)
		assert.NotContains(t, result, "templates") // Templates should NOT use base path to avoid data retention
		assert.Contains(t, result, validTeamID)
		assert.Contains(t, result, validBoardID)
		assert.Contains(t, result, "validFile")
	})

	t.Run("Should allow global team ID for templates", func(t *testing.T) {
		result, err := getDestinationFilePath(true, "0", validBoardID, "template-file.jpg")
		assert.NoError(t, err)
		assert.Contains(t, result, "0")
		assert.Contains(t, result, validBoardID)
		assert.Contains(t, result, "template-file.jpg")
		assert.NotContains(t, result, "templates") // Templates use direct path to avoid data retention
	})

	t.Run("Should reject global team ID for non-templates for security", func(t *testing.T) {
		// Ensure we're not in test mode to verify production behavior
		origEnv := os.Getenv("FOCALBOARD_UNIT_TESTING")
		os.Unsetenv("FOCALBOARD_UNIT_TESTING")
		defer func() {
			if origEnv != "" {
				os.Setenv("FOCALBOARD_UNIT_TESTING", origEnv)
			}
		}()

		result, err := getDestinationFilePath(false, "0", validBoardID, "non-template-file.jpg")
		// Global team ID should now be rejected for non-template operations for security
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid teamID in ValidateTeamID")
		assert.Equal(t, "", result)
	})

	t.Run("Should allow valid template paths", func(t *testing.T) {
		validBoardID2 := "bxhwgf5r15fr3dryfozf1dmy41r" // Another valid 27-char board ID (fixed length)
		result, err := getDestinationFilePath(true, validTeamID, validBoardID2, "file.jpg")
		assert.NoError(t, err)
		assert.Contains(t, result, validTeamID)
		assert.Contains(t, result, validBoardID2)
		assert.Contains(t, result, "file.jpg")
		assert.NotContains(t, result, "templates") // Templates use direct path to avoid data retention
	})

	t.Run("Should not affect non-template files", func(t *testing.T) {
		validBoardID3 := "b12345678901234567890123456" // Another valid 27-char board ID
		result, err := getDestinationFilePath(false, validTeamID, validBoardID3, "filename")
		assert.NoError(t, err)
		assert.NotContains(t, result, "templates")
		assert.NotContains(t, result, validTeamID) // Non-templates don't include teamID in path
		assert.Contains(t, result, validBoardID3)  // boardID is included for ownership tracking
		assert.Contains(t, result, "filename")
	})

	t.Run("Should reject absolute paths in teamID", func(t *testing.T) {
		result, err := getDestinationFilePath(true, "/plugins/file.tar.gz", validBoardID, "filename")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid teamID in ValidateTeamID")
		assert.Equal(t, "", result)
	})

	t.Run("Should reject absolute paths in boardID", func(t *testing.T) {
		result, err := getDestinationFilePath(false, validTeamID, "/usr/bin/executable", "filename")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid boardID")
		assert.Equal(t, "", result)
	})
}

func TestValidatePathComponent(t *testing.T) {
	t.Run("Should allow valid components", func(t *testing.T) {
		validComponents := []string{
			"team123",
			"board_456",
			"file.jpg",
			"valid-name",
			"123abc",
		}

		for _, component := range validComponents {
			err := validatePathComponent(component)
			assert.NoError(t, err, "Component should be valid: %s", component)
		}
	})

	t.Run("Should reject path traversal attempts", func(t *testing.T) {
		invalidComponents := []string{
			"../etc",
			"../../passwd",
			"../../../root",
			"dir/../other",
			"..\\windows",
		}

		for _, component := range invalidComponents {
			err := validatePathComponent(component)
			assert.Error(t, err, "Component should be invalid: %s", component)
		}
	})

	t.Run("Should reject absolute paths", func(t *testing.T) {
		invalidComponents := []string{
			"/etc/passwd",
			"\\windows\\system32",
			"/root",
		}

		for _, component := range invalidComponents {
			err := validatePathComponent(component)
			assert.Error(t, err, "Component should be invalid: %s", component)
		}
	})

	t.Run("Should reject empty components", func(t *testing.T) {
		err := validatePathComponent("")
		assert.Error(t, err)
	})

	t.Run("Should reject components with invalid characters", func(t *testing.T) {
		invalidComponents := []string{
			"file with spaces",
			"file|pipe",
			"file<redirect",
			"file>redirect",
			"file&command",
			"file;command",
		}

		for _, component := range invalidComponents {
			err := validatePathComponent(component)
			assert.Error(t, err, "Component should be invalid: %s", component)
		}
	})

	t.Run("Should reject absolute path components", func(t *testing.T) {
		absolutePaths := []string{
			"/plugins/file.tar.gz",
			"/plugins/library.so",
			"/etc/passwd",
			"/usr/bin/executable",
			"\\windows\\system32\\file.exe",
			"/var/www/html/script.php",
		}

		for _, component := range absolutePaths {
			err := validatePathComponent(component)
			assert.Error(t, err, "Invalid path component should be rejected: %s", component)
		}
	})
}

func TestGlobalTemplateFilePathValidation(t *testing.T) {
	// This test reproduces the original error scenario that was happening during
	// global template initialization with team ID "0"
	t.Run("Should allow global team ID in template file operations", func(t *testing.T) {
		globalTeamID := "0"                           // model.GlobalTeamID
		validBoardID := "bbn1888mprfrm5fjw9f1je9x3xo" // Example board ID from the error
		filename := "76fwrj36hptg6dywka4k5mt3sph.png" // Example filename from the error

		// This should not return an error with our fix
		result, err := getDestinationFilePath(true, globalTeamID, validBoardID, filename)
		assert.NoError(t, err)
		assert.Contains(t, result, globalTeamID)
		assert.Contains(t, result, validBoardID)
		assert.Contains(t, result, filename)
	})
}

func TestUserCreatedTemplateFilePathValidation(t *testing.T) {
	// This test verifies that user-created templates with regular team IDs work correctly
	t.Run("Should allow regular team ID for user-created templates", func(t *testing.T) {
		userTeamID := "abcdefghijklmnopqrstuvwxyz" // Regular team ID for user-created template
		validBoardID := utils.NewID(utils.IDTypeBoard)
		filename := "user-template-image.png" // Template file

		// User-created templates should work with regular team IDs
		result, err := getDestinationFilePath(true, userTeamID, validBoardID, filename)
		assert.NoError(t, err)
		assert.Contains(t, result, userTeamID)
		assert.Contains(t, result, validBoardID)
		assert.Contains(t, result, filename)

		// Should use template path structure (not base path)
		assert.NotContains(t, result, "boards/")
		assert.Equal(t, userTeamID+"/"+validBoardID+"/"+filename, result)
	})
}

const (
	ownershipTeamID   = "validteamid1234567890123456"
	ownershipBoardID  = "bvalidboard1234567890123456"
	ownershipOtherID  = "botherboard1234567890123456"
	ownershipFilename = "7validfile1234567890123456.txt"
	ownershipFileID   = "validfile1234567890123456"
)

// fileRefBlock builds a block of the given type that references filename under the given field name.
func fileRefBlock(blockType model.BlockType, field, filename string) *model.Block {
	return &model.Block{
		ID:      "blockid1234567890123456789",
		BoardID: ownershipBoardID,
		Type:    blockType,
		Fields:  map[string]interface{}{field: filename},
	}
}

// legacyRefCase describes one shape of block reference that the ad-hoc scan has to recognise.
// Boards has written file references under two field names over time and either can appear on
// either block type, so all four combinations must resolve to the same answer.
type legacyRefCase struct {
	name             string
	imageBlocks      []*model.Block
	attachmentBlocks []*model.Block
	expectError      bool
}

func legacyRefCases(filename string) []legacyRefCase {
	return []legacyRefCase{
		{
			name:        "image block referencing the file through fileId",
			imageBlocks: []*model.Block{fileRefBlock(model.TypeImage, model.BlockFieldFileId, filename)},
		},
		{
			name:        "image block referencing the file through attachmentId",
			imageBlocks: []*model.Block{fileRefBlock(model.TypeImage, model.BlockFieldAttachmentId, filename)},
		},
		{
			// The shape the webapp actually writes for card attachments.
			name:             "attachment block referencing the file through fileId",
			attachmentBlocks: []*model.Block{fileRefBlock(model.TypeAttachment, model.BlockFieldFileId, filename)},
		},
		{
			name:             "attachment block referencing the file through attachmentId",
			attachmentBlocks: []*model.Block{fileRefBlock(model.TypeAttachment, model.BlockFieldAttachmentId, filename)},
		},
		{
			name: "attachment block carrying both field names",
			attachmentBlocks: []*model.Block{{
				ID:      "blockid1234567890123456789",
				BoardID: ownershipBoardID,
				Type:    model.TypeAttachment,
				Fields: map[string]interface{}{
					model.BlockFieldFileId:       filename,
					model.BlockFieldAttachmentId: filename,
				},
			}},
		},
		{
			name:             "blocks referencing a different file",
			imageBlocks:      []*model.Block{fileRefBlock(model.TypeImage, model.BlockFieldFileId, "7otherfile123456789012345.txt")},
			attachmentBlocks: []*model.Block{fileRefBlock(model.TypeAttachment, model.BlockFieldAttachmentId, "7otherfile123456789012345.txt")},
			expectError:      true,
		},
		{
			name:        "board has no image or attachment blocks",
			expectError: true,
		},
		{
			name:        "block field holds a non-string value",
			imageBlocks: []*model.Block{{ID: "blockid1234567890123456789", BoardID: ownershipBoardID, Type: model.TypeImage, Fields: map[string]interface{}{model.BlockFieldFileId: 42}}},
			expectError: true,
		},
		{
			name:        "block has no fields at all",
			imageBlocks: []*model.Block{{ID: "blockid1234567890123456789", BoardID: ownershipBoardID, Type: model.TypeImage, Fields: map[string]interface{}{}}},
			expectError: true,
		},
	}
}

// expectBlockScan primes the two queries validateFileReferencedByBoard always issues.
func expectBlockScan(th *TestHelper, imageBlocks, attachmentBlocks []*model.Block) {
	th.Store.EXPECT().GetBlocksWithType(ownershipBoardID, model.TypeImage).Return(imageBlocks, nil)
	th.Store.EXPECT().GetBlocksWithType(ownershipBoardID, model.TypeAttachment).Return(attachmentBlocks, nil)
}

func TestValidateFileReferencedByBoard(t *testing.T) {
	for _, tc := range legacyRefCases(ownershipFilename) {
		t.Run(tc.name, func(t *testing.T) {
			th, _ := SetupTestHelper(t)
			expectBlockScan(th, tc.imageBlocks, tc.attachmentBlocks)

			err := th.App.validateFileReferencedByBoard(ownershipBoardID, ownershipFilename)
			if tc.expectError {
				assert.ErrorIs(t, err, ErrFileNotReferencedByBoard)
				return
			}
			assert.NoError(t, err)
		})
	}

	t.Run("Should propagate an error from the image block query without querying attachments", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		th.Store.EXPECT().GetBlocksWithType(ownershipBoardID, model.TypeImage).Return(nil, errDummy)

		err := th.App.validateFileReferencedByBoard(ownershipBoardID, ownershipFilename)
		assert.ErrorIs(t, err, errDummy)
	})

	t.Run("Should propagate an error from the attachment block query", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		th.Store.EXPECT().GetBlocksWithType(ownershipBoardID, model.TypeImage).Return([]*model.Block{}, nil)
		th.Store.EXPECT().GetBlocksWithType(ownershipBoardID, model.TypeAttachment).Return(nil, errDummy)

		err := th.App.validateFileReferencedByBoard(ownershipBoardID, ownershipFilename)
		assert.ErrorIs(t, err, errDummy)
	})
}

func TestValidateFileOwnership(t *testing.T) {
	validTeamID := ownershipTeamID
	validBoardID := ownershipBoardID
	otherBoardID := ownershipOtherID
	filename := ownershipFilename

	// Paths that carry the owning board, so ownership resolves without scanning blocks.
	t.Run("Should allow a template file whose path matches the board", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: filepath.Join(validTeamID, validBoardID, filename),
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should allow access when boardID in path matches", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: "boards/20260317/" + validBoardID + "/" + filename,
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should deny access when boardID in path belongs to a different board", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: "boards/20260317/" + otherBoardID + "/" + filename,
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not belong to the specified board")
	})

	// Legacy uploads (boards/YYYYMMDD/filename, written before v9.2.4) have no board in the
	// path, so every reference shape has to be recognised by the block scan instead.
	for _, tc := range legacyRefCases(filename) {
		t.Run("Legacy path, "+tc.name, func(t *testing.T) {
			th, _ := SetupTestHelper(t)
			fileInfo := &mm_model.FileInfo{
				Id:   ownershipFileID,
				Path: "boards/20240618/" + filename,
			}
			th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
			expectBlockScan(th, tc.imageBlocks, tc.attachmentBlocks)

			err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "file does not belong to the specified board")
				return
			}
			assert.NoError(t, err)
		})
	}

	// A template path belonging to another board is not conclusive on the read path: the file
	// may still have been shared into this board, so the scan gets the final say.
	t.Run("Should fall back to the block scan when a template path names a different board", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: filepath.Join(validTeamID, otherBoardID, filename),
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
		expectBlockScan(th, []*model.Block{}, []*model.Block{fileRefBlock(model.TypeAttachment, model.BlockFieldFileId, filename)})

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should deny a template path naming a different board that no block references", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: filepath.Join(validTeamID, otherBoardID, filename),
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
		expectBlockScan(th, []*model.Block{}, []*model.Block{})

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not belong to the specified board")
	})

	t.Run("Should allow access to legacy file (empty PostId) referenced by board", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:     ownershipFileID,
			PostId: "", // legacy file — no board recorded
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
		expectBlockScan(th, []*model.Block{fileRefBlock(model.TypeImage, model.BlockFieldFileId, filename)}, []*model.Block{})

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should fall back to the block scan when the stored path is the empty sentinel", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{Id: ownershipFileID, Path: emptyString}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
		expectBlockScan(th, []*model.Block{}, []*model.Block{fileRefBlock(model.TypeAttachment, model.BlockFieldFileId, filename)})

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should propagate store errors raised by the block scan", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{Id: ownershipFileID, Path: "boards/20240618/" + filename}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
		th.Store.EXPECT().GetBlocksWithType(validBoardID, model.TypeImage).Return(nil, errDummy)

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.ErrorIs(t, err, errDummy)
	})

	t.Run("Should handle file info not found", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(nil, model.NewErrNotFound("file not found"))

		err := th.App.ValidateFileOwnership(validTeamID, validBoardID, filename)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestValidateFileOwnershipForBlockWrite(t *testing.T) {
	validTeamID := ownershipTeamID
	validBoardID := ownershipBoardID
	otherBoardID := ownershipOtherID
	filename := ownershipFilename

	t.Run("Should allow file whose boardID in path matches", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: "boards/20260317/" + validBoardID + "/" + filename,
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)

		err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should reject file whose boardID in path belongs to a different board", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: "boards/20260317/" + otherBoardID + "/" + filename,
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)

		err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not belong to the specified board")
	})

	t.Run("Should allow template file whose path matches the board", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: filepath.Join(validTeamID, validBoardID, filename),
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)

		err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should reject template file whose path belongs to a different board", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{
			Id:   ownershipFileID,
			Path: filepath.Join(validTeamID, otherBoardID, filename),
		}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)

		err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not belong to the specified board")
	})

	// Attaching a legacy file to a block is only allowed when this board already references it,
	// which is what keeps another team's file from being pulled in by ID (MM-67760).
	for _, tc := range legacyRefCases(filename) {
		t.Run("Legacy path, "+tc.name, func(t *testing.T) {
			th, _ := SetupTestHelper(t)
			fileInfo := &mm_model.FileInfo{
				Id:   ownershipFileID,
				Path: "boards/20260317/" + filename, // old format: no boardID in path
			}
			th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
			expectBlockScan(th, tc.imageBlocks, tc.attachmentBlocks)

			err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "file does not belong to the specified board")
				return
			}
			assert.NoError(t, err)
		})
	}

	t.Run("Should propagate store errors raised by the block scan", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		fileInfo := &mm_model.FileInfo{Id: ownershipFileID, Path: "boards/20260317/" + filename}
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(fileInfo, nil)
		th.Store.EXPECT().GetBlocksWithType(validBoardID, model.TypeImage).Return(nil, errDummy)

		err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
		assert.ErrorIs(t, err, errDummy)
	})

	t.Run("Should allow file with no FileInfo record (very old upload)", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(nil, model.NewErrNotFound("file not found"))

		err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})

	t.Run("Should allow file when the store returns no FileInfo and no error", func(t *testing.T) {
		th, _ := SetupTestHelper(t)
		th.Store.EXPECT().GetFileInfo(ownershipFileID).Return(nil, nil)

		err := th.App.validateFileOwnershipForBlockWrite(validTeamID, validBoardID, filename)
		assert.NoError(t, err)
	})
}

func TestGetFilePathWithGlobalTeamID(t *testing.T) {
	th, _ := SetupTestHelper(t)

	t.Run("when TeamID is GlobalTeamID and board is template", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(nil, nil)
		th.Store.EXPECT().GetBoard("bvalidboard1234567890123456").Return(&model.Board{
			ID:         "bvalidboard1234567890123456",
			TeamID:     model.GlobalTeamID,
			IsTemplate: true,
		}, nil)

		// Mock FileExists calls for GlobalTeamID template file path checking
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", mock.Anything).Return(false, nil).Twice()

		fileInfo, filePath, err := th.App.GetFilePath(model.GlobalTeamID, "bvalidboard1234567890123456", "7fileInfoID.txt")
		assert.NoError(t, err)
		assert.Nil(t, fileInfo)
		assert.Equal(t, "0/bvalidboard1234567890123456/7fileInfoID.txt", filePath)
	})

	t.Run("when TeamID is GlobalTeamID and board is not template", func(t *testing.T) {
		// Ensure we're not in test mode to verify production security behavior
		origEnv := os.Getenv("FOCALBOARD_UNIT_TESTING")
		os.Unsetenv("FOCALBOARD_UNIT_TESTING")
		defer func() {
			if origEnv != "" {
				os.Setenv("FOCALBOARD_UNIT_TESTING", origEnv)
			}
		}()

		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(nil, nil)
		th.Store.EXPECT().GetBoard("bvalidboard1234567890123456").Return(&model.Board{
			ID:         "bvalidboard1234567890123456",
			TeamID:     model.GlobalTeamID,
			IsTemplate: false,
		}, nil)

		fileInfo, filePath, err := th.App.GetFilePath(model.GlobalTeamID, "bvalidboard1234567890123456", "7fileInfoID.txt")
		// GlobalTeamID ("0") should be rejected for non-template file operations for security
		// This prevents path traversal attacks
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid teamID in ValidateTeamID")
		assert.Nil(t, fileInfo)
		assert.Equal(t, "", filePath)
	})
}
