package sqlstore

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"github.com/mattermost/mattermost-plugin-boards/server/model"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (s *SQLStore) saveFileInfo(db sq.BaseRunner, fileInfo *mmModel.FileInfo) error {
	query := s.getQueryBuilder(db).
		Insert(s.tablePrefix+"file_info").
		Columns(
			"id",
			"create_at",
			"name",
			"extension",
			"size",
			"delete_at",
			"path",
			"archived",
		).
		Values(
			fileInfo.Id,
			fileInfo.CreateAt,
			fileInfo.Name,
			fileInfo.Extension,
			fileInfo.Size,
			fileInfo.DeleteAt,
			fileInfo.Path,
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

func (s *SQLStore) getFileInfo(db sq.BaseRunner, id string) (*mmModel.FileInfo, error) {
	query := s.getQueryBuilder(db).
		Select(
			"id",
			"create_at",
			"delete_at",
			"name",
			"extension",
			"size",
			"archived",
			"path",
		).
		From(s.tablePrefix + "file_info").
		Where(sq.Eq{"Id": id})

	row := query.QueryRow()

	fileInfo := mmModel.FileInfo{}

	err := row.Scan(
		&fileInfo.Id,
		&fileInfo.CreateAt,
		&fileInfo.DeleteAt,
		&fileInfo.Name,
		&fileInfo.Extension,
		&fileInfo.Size,
		&fileInfo.Archived,
		&fileInfo.Path,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.NewErrNotFound("file info ID=" + id)
		}

		s.logger.Error("error scanning fileinfo row", mlog.String("id", id), mlog.Err(err))
		return nil, err
	}

	return &fileInfo, nil
}
