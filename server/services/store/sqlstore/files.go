// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"errors"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-boards/server/model"
	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (s *SQLStore) getFileInfo(_ sq.BaseRunner, id string) (*mmModel.FileInfo, error) {
	fileInfo, err := s.servicesAPI.GetFileInfo(id)
	if err != nil {
		// Not finding fileinfo is fine because we don't have data for
		// any existing files already uploaded in Boards before this code
		// was deployed.
		var appErr *mmModel.AppError
		if errors.As(err, &appErr) {
			if appErr.StatusCode == http.StatusNotFound {
				return nil, model.NewErrNotFound("file info ID=" + id)
			}
		}

		s.logger.Error("error fetching fileinfo",
			mlog.String("id", id),
			mlog.Err(err),
		)
		return nil, err
	}

	return fileInfo, nil
}

func (s *SQLStore) saveFileInfo(db sq.BaseRunner, fileInfo *mmModel.FileInfo) error {
	query := s.getQueryBuilder(db).
		Insert("FileInfo").
		Columns(
			"Id",
			"CreatorId",
			"PostId",
			"CreateAt",
			"UpdateAt",
			"DeleteAt",
			"Path",
			"ThumbnailPath",
			"PreviewPath",
			"Name",
			"Extension",
			"Size",
			"MimeType",
			"Width",
			"Height",
			"HasPreviewImage",
			"MiniPreview",
			"Content",
			"RemoteId",
			"Archived",
		).
		Values(
			fileInfo.Id,
			fileInfo.CreatorId,
			fileInfo.PostId,
			fileInfo.CreateAt,
			fileInfo.UpdateAt,
			fileInfo.DeleteAt,
			fileInfo.Path,
			fileInfo.ThumbnailPath,
			fileInfo.PreviewPath,
			fileInfo.Name,
			fileInfo.Extension,
			fileInfo.Size,
			fileInfo.MimeType,
			fileInfo.Width,
			fileInfo.Height,
			fileInfo.HasPreviewImage,
			fileInfo.MiniPreview,
			fileInfo.Content,
			fileInfo.RemoteId,
			false,
		)

	if _, err := query.Exec(); err != nil {
		s.logger.Error(
			"failed to save fileinfo",
			mlog.String("file_name", fileInfo.Name),
			mlog.Int("size", fileInfo.Size),
			mlog.Err(err),
		)
		return err
	}

	return nil
}

func (s *SQLStore) restoreFiles(db sq.BaseRunner, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	query := s.getQueryBuilder(db).
		Update("FileInfo").
		Set("DeleteAt", 0).
		Where(sq.Eq{"Id": fileIDs})

	if _, err := query.Exec(); err != nil {
		s.logger.Error(
			"failed to restore files",
			mlog.Int("file_count", len(fileIDs)),
			mlog.Err(err),
		)
		return err
	}

	return nil
}
