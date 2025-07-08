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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	mm_model "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore/mocks"
)

const (
	testFileName = "temp-file-name"
	testBoardID  = "btestboardid1234567890123456"
	testPath     = "/path/to/file/fileName.txt"
)

var errDummy = errors.New("hello")

type TestError struct{}

func (err *TestError) Error() string { return "Mocked File backend error" }

func TestGetFileReader(t *testing.T) {
	validTeamID := "abcdefghijklmnopqrstuvwxyz" // 26 chars - valid Mattermost ID
	testFilePath := filepath.Join(validTeamID, "test-board-id", "temp-file-name")

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
		filePath := filepath.Join("0", "test-board-id", "temp-file-name")
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

		mockedFileBackend.On("FileExists", filePath).Return(fileExistsFunc, fileExistsErrorFunc)
		mockedFileBackend.On("FileExists", testFileName).Return(fileExistsFunc, fileExistsErrorFunc)
		mockedFileBackend.On("MoveFile", testFileName, filePath).Return(moveFileFunc)
		mockedFileBackend.On("Reader", filePath).Return(readerFunc, readerErrorFunc)

		actual, _ := th.App.GetFileReader(workspaceid, testBoardID, testFileName)
		assert.Equal(t, mockedReadCloseSeek, actual)
	})

	t.Run("should return file reader, if file doesnot exists in new filepath and old file path", func(t *testing.T) {
		filePath := filepath.Join("0", "test-board-id", "temp-file-name")
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
	t.Run("should save file to file store using file backend", func(t *testing.T) {
		fileName := "temp-file-name.txt"
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		th.Store.EXPECT().SaveFileInfo(gomock.Any()).Return(nil)

		writeFileFunc := func(reader io.Reader, path string) int64 {
			paths := strings.Split(path, string(os.PathSeparator))
			assert.Equal(t, "boards", paths[0])
			assert.Equal(t, time.Now().Format("20060102"), paths[1])
			fileName = paths[2]
			return int64(10)
		}

		writeFileErrorFunc := func(reader io.Reader, filePath string) error {
			return nil
		}

		mockedFileBackend.On("WriteFile", mockedReadCloseSeek, mock.Anything).Return(writeFileFunc, writeFileErrorFunc)
		actual, err := th.App.SaveFile(mockedReadCloseSeek, "1", testBoardID, fileName, false)
		assert.Equal(t, fileName, actual)
		assert.Nil(t, err)
	})

	t.Run("should save .jpeg file as jpg file to file store using file backend", func(t *testing.T) {
		fileName := "temp-file-name.jpeg"
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		th.Store.EXPECT().SaveFileInfo(gomock.Any()).Return(nil)

		writeFileFunc := func(reader io.Reader, path string) int64 {
			paths := strings.Split(path, string(os.PathSeparator))
			assert.Equal(t, "boards", paths[0])
			assert.Equal(t, time.Now().Format("20060102"), paths[1])
			assert.Equal(t, "jpg", strings.Split(paths[2], ".")[1])
			return int64(10)
		}

		writeFileErrorFunc := func(reader io.Reader, filePath string) error {
			return nil
		}

		mockedFileBackend.On("WriteFile", mockedReadCloseSeek, mock.Anything).Return(writeFileFunc, writeFileErrorFunc)
		actual, err := th.App.SaveFile(mockedReadCloseSeek, "1", "test-board-id", fileName, false)
		assert.Nil(t, err)
		assert.NotNil(t, actual)
	})

	t.Run("should return error when fileBackend.WriteFile returns error", func(t *testing.T) {
		fileName := "temp-file-name.jpeg"
		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedError := &TestError{}

		writeFileFunc := func(reader io.Reader, path string) int64 {
			paths := strings.Split(path, string(os.PathSeparator))
			assert.Equal(t, "boards", paths[0])
			assert.Equal(t, time.Now().Format("20060102"), paths[1])
			assert.Equal(t, "jpg", strings.Split(paths[2], ".")[1])
			return int64(10)
		}

		writeFileErrorFunc := func(reader io.Reader, filePath string) error {
			return mockedError
		}

		mockedFileBackend.On("WriteFile", mockedReadCloseSeek, mock.Anything).Return(writeFileFunc, writeFileErrorFunc)
		actual, err := th.App.SaveFile(mockedReadCloseSeek, "1", "test-board-id", fileName, false)
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
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: testPath,
		}, nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedReadCloseSeek := &mocks.ReadCloseSeeker{}
		readerFunc := func(path string) filestore.ReadCloseSeeker {
			return mockedReadCloseSeek
		}

		readerErrorFunc := func(path string) error {
			return nil
		}
		mockedFileBackend.On("Reader", testPath).Return(readerFunc, readerErrorFunc)
		mockedFileBackend.On("FileExists", testPath).Return(true, nil)

		validBoardID := "bvalidboard1234567890123456"
		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.NoError(t, err)
		assert.NotNil(t, fileInfo)
		assert.NotNil(t, seeker)
	})

	t.Run("when GetFilePath() throws error", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(nil, errDummy)

		validBoardID := "bvalidboard1234567890123456"
		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.Error(t, err)
		assert.Nil(t, fileInfo)
		assert.Nil(t, seeker)
	})

	t.Run("when FileExists returns false", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: testPath,
		}, nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("FileExists", testPath).Return(false, nil)

		validBoardID := "bvalidboard1234567890123456"
		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.Error(t, err)
		assert.Nil(t, fileInfo)
		assert.Nil(t, seeker)
	})
	t.Run("when FileReader throws error", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(&mm_model.FileInfo{
			Id:   "fileInfoID",
			Path: testPath,
		}, nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("Reader", testPath).Return(nil, errDummy)
		mockedFileBackend.On("FileExists", testPath).Return(true, nil)

		validBoardID := "bvalidboard1234567890123456"
		fileInfo, seeker, err := th.App.GetFile(validTeamID, validBoardID, "7fileInfoID.txt")
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
		validBoardID := "bvalidboard1234567890123456"
		fileInfo, filePath, err := th.App.GetFilePath(validTeamID, validBoardID, "7fileInfoID.txt")
		assert.NoError(t, err)
		assert.NotNil(t, fileInfo)
		assert.Equal(t, testPath, filePath)
	})

	t.Run("when FileInfo doesn't exist", func(t *testing.T) {
		th.Store.EXPECT().GetFileInfo("fileInfoID").Return(nil, nil)

		validTeamID := "abcdefghijklmnopqrstuvwxyz" // 26 chars - valid Mattermost ID
		validBoardID := "bvalidboard1234567890123456"
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
		validBoardID := "bvalidboard1234567890123456"
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
		Fields:     map[string]interface{}{"fileId": "7fileName.jpg"},
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
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo("fileName").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		updatedFileNames, err := th.App.CopyCardFiles(validTestBoardID, []*model.Block{imageBlock}, false)
		assert.NoError(t, err)
		assert.Equal(t, "7fileName.jpg", imageBlock.Fields["fileId"])
		assert.NotNil(t, updatedFileNames["7fileName.jpg"])
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
			Fields:     map[string]interface{}{"fileId": "7fileName.jpg"},
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
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo("fileName").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		updatedFileNames, err := th.App.CopyCardFiles(validTestBoardID, []*model.Block{attachmentBlock}, false)
		assert.NoError(t, err)
		assert.NotNil(t, updatedFileNames[imageBlock.Fields["fileId"].(string)])
	})

	t.Run("Board exists, image block, without FileInfo", func(t *testing.T) {
		th.Store.EXPECT().GetBoard("boardID").Return(&model.Board{
			ID:         "boardID",
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo(gomock.Any()).Return(nil, nil)
		th.Store.EXPECT().SaveFileInfo(gomock.Any()).Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		updatedFileNames, err := th.App.CopyCardFiles("boardID", []*model.Block{imageBlock}, false)
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
		Fields:     map[string]interface{}{"fileId": "7fileName1234567890987654321.jpg"},
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
			IsTemplate: false,
		}, nil)
		th.Store.EXPECT().GetFileInfo("fileName").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)
		th.Store.EXPECT().PatchBlocks(gomock.Any(), "userID").Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
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
		th.Store.EXPECT().GetBoard(validTestBoardID2).Return(&model.Board{ID: validTestBoardID2, IsTemplate: false}, nil)
		th.Store.EXPECT().GetFileInfo("xhwgf5r15fr3dryfozf1dmy41r").Return(fileInfo, nil)
		th.Store.EXPECT().SaveFileInfo(fileInfo).Return(nil)
		th.Store.EXPECT().PatchBlocks(gomock.Any(), "userID").Return(nil)

		mockedFileBackend := &mocks.FileBackend{}
		th.App.filesBackend = mockedFileBackend
		mockedFileBackend.On("CopyFile", mock.Anything, mock.Anything).Return(nil)

		err := th.App.CopyAndUpdateCardFiles(validTestBoardID2, "userID", []*model.Block{validImageBlock}, false)
		assert.NoError(t, err)
	})

	t.Run("Invalid file ID length", func(t *testing.T) {
		th.Store.EXPECT().GetBoard(validTestBoardID2).Return(&model.Board{ID: validTestBoardID2, IsTemplate: false}, nil)
		err := th.App.CopyAndUpdateCardFiles(validTestBoardID2, "userID", []*model.Block{invalidShortFileIDBlock}, false)
		assert.ErrorIs(t, err, model.NewErrBadRequest("Invalid Block ID"))
	})

	t.Run("Empty file ID", func(t *testing.T) {
		th.Store.EXPECT().GetBoard(validTestBoardID2).Return(&model.Board{ID: validTestBoardID2, IsTemplate: false}, nil)
		err := th.App.CopyAndUpdateCardFiles(validTestBoardID2, "userID", []*model.Block{emptyFileBlock}, false)
		assert.ErrorIs(t, err, model.NewErrBadRequest("Block ID cannot be empty"))
	})
}

func TestCopyCardFiles(t *testing.T) {
	app := &App{}

	t.Run("ValidFileID", func(t *testing.T) {
		sourceBoardID := "bsourceboard123456789012345"
		destBoardID := "bdestinationboard1234567890"
		copiedBlocks := []*model.Block{
			{
				Type:    model.TypeImage,
				Fields:  map[string]interface{}{"fileId": "validFileID"},
				BoardID: destBoardID,
			},
		}

		newFileNames, err := app.CopyCardFiles(sourceBoardID, copiedBlocks, false)

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

		newFileNames, err := app.CopyCardFiles(sourceBoardID, copiedBlocks, false)

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

	t.Run("Should allow global team ID for non-templates", func(t *testing.T) {
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
		assert.NotContains(t, result, validTeamID)   // Non-templates don't include teamID in path
		assert.NotContains(t, result, validBoardID3) // Non-templates don't include boardID in path
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
		userTeamID := "abcdefghijklmnopqrstuvwxyz"    // Regular team ID for user-created template
		validBoardID := "bvalidboard1234567890123456" // Valid 27-char board ID starting with 'b' (same format as other tests)
		filename := "user-template-image.png"         // Template file

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
