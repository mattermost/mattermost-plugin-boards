// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	mm_model "github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-boards/server/utils"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
)

const emptyString = "empty"

var errEmptyFilename = errors.New("IsFileArchived: empty filename not allowed")
var ErrFileNotFound = errors.New("file not found")
var ErrFileNotReferencedByBoard = errors.New("file not referenced by board")

func (a *App) SaveFile(reader io.Reader, teamID, boardID, filename string, asTemplate bool) (string, error) {
	// NOTE: File extension includes the dot
	fileExtension := strings.ToLower(filepath.Ext(filename))
	if fileExtension == ".jpeg" {
		fileExtension = ".jpg"
	}

	createdFilename := utils.NewID(utils.IDTypeNone)
	newFileName := fmt.Sprintf(`%s%s`, createdFilename, fileExtension)
	if asTemplate {
		newFileName = filename
	}
	filePath, pathErr := getDestinationFilePath(asTemplate, teamID, boardID, newFileName)
	if pathErr != nil {
		return "", fmt.Errorf("invalid file path parameters: %w", pathErr)
	}

	fileSize, appErr := a.filesBackend.WriteFile(reader, filePath)
	if appErr != nil {
		return "", fmt.Errorf("unable to store the file in the files storage: %w", appErr)
	}

	fileInfo := model.NewFileInfo(filename)
	fileInfo.Id = getFileInfoID(createdFilename)
	fileInfo.Path = filePath
	fileInfo.Size = fileSize

	err := a.store.SaveFileInfo(fileInfo)
	if err != nil {
		return "", err
	}

	return newFileName, nil
}

func (a *App) GetFileInfo(filename string) (*mm_model.FileInfo, error) {
	if len(filename) == 0 {
		return nil, errEmptyFilename
	}

	// filename is in the format 7<some-alphanumeric-string>.<extension>
	// we want to extract the <some-alphanumeric-string> part of this as this
	// will be the fileinfo id.
	parts := strings.Split(filename, ".")
	if len(parts) < 1 || len(parts[0]) <= 1 {
		return nil, model.NewErrNotFound("No file found with name" + filename)
	}

	fileInfoID := getFileInfoID(parts[0])

	if (fileInfoID) == "" {
		return nil, model.NewErrNotFound("No file found with name" + filename)
	}

	fileInfo, err := a.store.GetFileInfo(fileInfoID)
	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}

// ValidateFileOwnership checks if a file belongs to the specified board and team.
func (a *App) ValidateFileOwnership(teamID, boardID, filename string) error {
	fileInfo, err := a.GetFileInfo(filename)
	if err != nil {
		return err
	}
	if fileInfo != nil && fileInfo.Path != "" && fileInfo.Path != emptyString {
		expectedPath := filepath.Join(teamID, boardID, filename)
		if fileInfo.Path == expectedPath {
			return nil
		}
		if err := a.validateFileReferencedByBoard(boardID, filename); err != nil {
			return model.NewErrPermission("file does not belong to the specified board")
		}
	} else {
		if err := a.validateFileReferencedByBoard(boardID, filename); err != nil {
			return model.NewErrPermission("file does not belong to the specified board")
		}
	}
	return nil
}

// validateFileReferencedByBoard checks if a file is referenced by blocks in the specified board.
func (a *App) validateFileReferencedByBoard(boardID, filename string) error {
	blocks, err := a.GetBlocksForBoard(boardID)
	if err != nil {
		return err
	}

	for _, block := range blocks {
		if block.Type == model.TypeImage || block.Type == model.TypeAttachment {
			if fileID, ok := block.Fields[model.BlockFieldFileId].(string); ok && fileID == filename {
				return nil
			}
			if attachmentID, ok := block.Fields[model.BlockFieldAttachmentId].(string); ok && attachmentID == filename {
				return nil
			}
		}
	}

	return fmt.Errorf("%w: file %s is not referenced by any block in board %s", ErrFileNotReferencedByBoard, filename, boardID)
}

func (a *App) GetFile(teamID, boardID, fileName string) (*mm_model.FileInfo, filestore.ReadCloseSeeker, error) {
	if err := a.ValidateFileOwnership(teamID, boardID, fileName); err != nil {
		a.logger.Error("GetFile: File ownership validation failed",
			mlog.String("Team", teamID),
			mlog.String("board", boardID),
			mlog.String("filename", fileName),
			mlog.Err(err))
		return nil, nil, err
	}

	fileInfo, filePath, err := a.GetFilePath(teamID, boardID, fileName)
	if err != nil {
		a.logger.Error("GetFile: Failed to GetFilePath.", mlog.String("Team", teamID), mlog.String("board", boardID), mlog.String("filename", fileName), mlog.Err(err))
		return nil, nil, err
	}

	exists, err := a.filesBackend.FileExists(filePath)
	if err != nil {
		a.logger.Error("GetFile: Failed to check if file exists as path. ", mlog.String("Path", filePath), mlog.Err(err))
		return nil, nil, err
	}

	if !exists {
		return nil, nil, ErrFileNotFound
	}

	reader, err := a.filesBackend.Reader(filePath)
	if err != nil {
		a.logger.Error("GetFile: Failed to get file reader of existing file at path", mlog.String("Path", filePath), mlog.Err(err))
		return nil, nil, err
	}

	return fileInfo, reader, nil
}

func (a *App) GetFilePath(teamID, boardID, fileName string) (*mm_model.FileInfo, string, error) {
	fileInfo, err := a.GetFileInfo(fileName)
	if err != nil && !model.IsErrNotFound(err) {
		return nil, "", err
	}

	var filePath string

	if fileInfo != nil && fileInfo.Path != "" && fileInfo.Path != emptyString {
		filePath = fileInfo.Path
	} else {
		// Validate path components to ensure proper file path handling
		if !mm_model.IsValidId(teamID) {
			return nil, "", fmt.Errorf("GetFilePath: invalid team ID for teamID %s", teamID) //nolint:err113
		}
		if err := model.IsValidId(boardID); err != nil {
			return nil, "", fmt.Errorf("invalid rootID in GetFilePath: %w", err)
		}
		if err := validatePathComponent(fileName); err != nil {
			return nil, "", fmt.Errorf("invalid fileName in GetFilePath: %w", err)
		}
		filePath = filepath.Join(teamID, boardID, fileName)
	}

	return fileInfo, filePath, nil
}

func getDestinationFilePath(isTemplate bool, teamID, boardID, filename string) (string, error) {
	// Validate inputs to ensure proper file path handling
	if !mm_model.IsValidId(teamID) {
		return "", fmt.Errorf("getDestinationFilePath: invalid team ID for teamID %s", teamID) //nolint:err113
	}
	if err := model.IsValidId(boardID); err != nil {
		return "", fmt.Errorf("invalid boardID: %w", err)
	}
	if err := validatePathComponent(filename); err != nil {
		return "", fmt.Errorf("invalid filename: %w", err)
	}

	// if saving a file for a template, save using the "old method" that is /teamID/boardID/fileName
	// this will prevent template files from being deleted by DataRetention,
	// which deletes all files inside the "date" subdirectory
	if isTemplate {
		return filepath.Join(teamID, boardID, filename), nil
	}
	return filepath.Join(utils.GetBaseFilePath(), filename), nil
}

// validatePathComponent ensures a path component contains only valid characters.
func validatePathComponent(component string) error {
	// This regex allows alphanumeric, hyphens, underscores, and dots
	// Empty strings and invalid characters (including path separators) are rejected
	validComponent := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validComponent.MatchString(component) {
		return fmt.Errorf("invalid path component: %s", component) //nolint:err113
	}

	return nil
}

func getFileInfoID(fileName string) string {
	// Boards ids are 27 characters long with a prefix character.
	// removing the prefix, returns the 26 character uuid
	return fileName[1:]
}

func (a *App) GetFileReader(teamID, boardID, filename string) (filestore.ReadCloseSeeker, error) {
	// Validate path components to ensure proper file path handling
	if !mm_model.IsValidId(teamID) {
		return nil, fmt.Errorf("GetFileReader: invalid team ID for teamID %s", teamID) //nolint:err113
	}
	if err := model.IsValidId(boardID); err != nil {
		return nil, fmt.Errorf("invalid rootID in GetFileReader: %w", err)
	}
	if err := validatePathComponent(filename); err != nil {
		return nil, fmt.Errorf("invalid filename in GetFileReader: %w", err)
	}

	filePath := filepath.Join(teamID, boardID, filename)
	exists, err := a.filesBackend.FileExists(filePath)
	if err != nil {
		return nil, err
	}
	// FIXUP: Check the deprecated old location
	if teamID == "0" && !exists {
		oldExists, err2 := a.filesBackend.FileExists(filename)
		if err2 != nil {
			return nil, err2
		}
		if oldExists {
			err2 := a.filesBackend.MoveFile(filename, filePath)
			if err2 != nil {
				a.logger.Error("ERROR moving file",
					mlog.String("old", filename),
					mlog.String("new", filePath),
					mlog.Err(err2),
				)
			} else {
				a.logger.Debug("Moved file",
					mlog.String("old", filename),
					mlog.String("new", filePath),
				)
			}
		}
	} else if !exists {
		return nil, ErrFileNotFound
	}

	reader, err := a.filesBackend.Reader(filePath)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (a *App) MoveFile(channelID, teamID, boardID, filename string) error {
	// Validate path components to ensure proper file path handling
	if !mm_model.IsValidId(channelID) {
		return fmt.Errorf("MoveFile: invalid channel ID for channelID %s", channelID) //nolint:err113
	}
	if !mm_model.IsValidId(teamID) {
		return fmt.Errorf("MoveFile: invalid team ID for teamID %s", teamID) //nolint:err113
	}
	if err := model.IsValidId(boardID); err != nil {
		return fmt.Errorf("invalid boardID in MoveFile: %w", err)
	}
	if err := validatePathComponent(filename); err != nil {
		return fmt.Errorf("invalid filename in MoveFile: %w", err)
	}

	oldPath := filepath.Join(channelID, boardID, filename)
	newPath := filepath.Join(teamID, boardID, filename)
	err := a.filesBackend.MoveFile(oldPath, newPath)
	if err != nil {
		a.logger.Error("ERROR moving file",
			mlog.String("old", oldPath),
			mlog.String("new", newPath),
			mlog.Err(err),
		)
		return err
	}
	return nil
}
func (a *App) CopyAndUpdateCardFiles(boardID, userID string, blocks []*model.Block, asTemplate bool) error {
	newFileNames, err := a.CopyCardFiles(boardID, blocks, asTemplate)
	if err != nil {
		a.logger.Error("Could not copy files while duplicating board", mlog.String("BoardID", boardID), mlog.Err(err))
	}

	// blocks now has updated file ids for any blocks containing files.  We need to update the database for them.
	blockIDs := make([]string, 0)
	blockPatches := make([]model.BlockPatch, 0)
	for _, block := range blocks {
		if block.Type == model.TypeImage || block.Type == model.TypeAttachment {
			if fileID, ok := block.Fields[model.BlockFieldFileId].(string); ok {
				if err = model.ValidateFileId(fileID); err == nil {
					blockIDs = append(blockIDs, block.ID)
					blockPatches = append(blockPatches, model.BlockPatch{
						UpdatedFields: map[string]interface{}{
							model.BlockFieldFileId: newFileNames[fileID],
						},
						DeletedFields: []string{model.BlockFieldAttachmentId},
					})
				} else {
					errMessage := fmt.Sprintf("invalid characters in block with key: %s, %s", block.Fields[model.BlockFieldFileId], err)
					return model.NewErrBadRequest(errMessage)
				}
			}

			if attachmentID, ok := block.Fields[model.BlockFieldAttachmentId].(string); ok {
				if err = model.ValidateFileId(attachmentID); err == nil {
					blockIDs = append(blockIDs, block.ID)
					blockPatches = append(blockPatches, model.BlockPatch{
						UpdatedFields: map[string]interface{}{
							model.BlockFieldAttachmentId: newFileNames[attachmentID],
						},
						DeletedFields: []string{model.BlockFieldFileId},
					})
				} else {
					errMessage := fmt.Sprintf("invalid characters in block with key: %s, %s", block.Fields[model.BlockFieldAttachmentId], err)
					return model.NewErrBadRequest(errMessage)
				}
			}
		}
	}
	a.logger.Debug("Duplicate boards patching file IDs", mlog.Int("count", len(blockIDs)))

	if len(blockIDs) != 0 {
		patches := &model.BlockPatchBatch{
			BlockIDs:     blockIDs,
			BlockPatches: blockPatches,
		}
		if err := a.store.PatchBlocks(patches, userID); err != nil {
			return fmt.Errorf("could not patch file IDs while duplicating board %s: %w", boardID, err)
		}
	}

	return nil
}

func (a *App) CopyCardFiles(sourceBoardID string, copiedBlocks []*model.Block, asTemplate bool) (map[string]string, error) {
	// Images attached in cards have a path comprising the card's board ID.
	// When we create a template from this board, we need to copy the files
	// with the new board ID in path.
	// Not doing so causing images in templates (and boards created from this
	// template) to fail to load.

	// look up ID of source sourceBoard, which may be different than the blocks.
	sourceBoard, err := a.GetBoard(sourceBoardID)
	if err != nil || sourceBoard == nil {
		return nil, fmt.Errorf("cannot fetch source board %s for CopyCardFiles: %w", sourceBoardID, err)
	}

	var destBoard *model.Board
	newFileNames := make(map[string]string)
	for _, block := range copiedBlocks {
		if block.Type != model.TypeImage && block.Type != model.TypeAttachment {
			continue
		}

		fileID, isOk := block.Fields["fileId"].(string)
		if !isOk {
			fileID, isOk = block.Fields["attachmentId"].(string)
			if !isOk {
				continue
			}
		}

		if err = model.ValidateFileId(fileID); err != nil {
			errMessage := fmt.Sprintf("Could not validate file ID while duplicating board with fileId: %s", fileID)
			return nil, model.NewErrBadRequest(errMessage)
		}

		// create unique filename
		ext := filepath.Ext(fileID)
		fileInfoID := utils.NewID(utils.IDTypeNone)
		destFilename := fileInfoID + ext

		if destBoard == nil || block.BoardID != destBoard.ID {
			destBoard = sourceBoard
			if block.BoardID != destBoard.ID {
				destBoard, err = a.GetBoard(block.BoardID)
				if err != nil {
					return nil, fmt.Errorf("cannot fetch destination board %s for CopyCardFiles: %w", sourceBoardID, err)
				}
			}
		}

		// GetFilePath will retrieve the correct path
		// depending on whether FileInfo table is used for the file.
		fileInfo, sourceFilePath, err := a.GetFilePath(sourceBoard.TeamID, sourceBoard.ID, fileID)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch destination board %s for CopyCardFiles: %w", sourceBoardID, err)
		}
		destinationFilePath, pathErr := getDestinationFilePath(asTemplate, destBoard.TeamID, destBoard.ID, destFilename)
		if pathErr != nil {
			return nil, fmt.Errorf("invalid destination file path: %w", pathErr)
		}

		if fileInfo == nil {
			fileInfo = model.NewFileInfo(destFilename)
		}
		fileInfo.Id = getFileInfoID(fileInfoID)
		fileInfo.Path = destinationFilePath
		err = a.store.SaveFileInfo(fileInfo)
		if err != nil {
			return nil, fmt.Errorf("CopyCardFiles: cannot create fileinfo: %w", err)
		}

		a.logger.Debug(
			"Copying card file",
			mlog.String("sourceFilePath", sourceFilePath),
			mlog.String("destinationFilePath", destinationFilePath),
		)

		if err := a.filesBackend.CopyFile(sourceFilePath, destinationFilePath); err != nil {
			a.logger.Error(
				"CopyCardFiles failed to copy file",
				mlog.String("sourceFilePath", sourceFilePath),
				mlog.String("destinationFilePath", destinationFilePath),
				mlog.Err(err),
			)
		}
		newFileNames[fileID] = destFilename
	}

	return newFileNames, nil
}
