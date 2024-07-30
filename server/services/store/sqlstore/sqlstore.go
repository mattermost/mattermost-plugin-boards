package sqlstore

import (
	"database/sql"
	"net/url"
	"strings"

	sq "github.com/Masterminds/squirrel"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"

	mmModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// SQLStore is a SQL database.
type SQLStore struct {
	db               *sql.DB
	dbType           string
	tablePrefix      string
	connectionString string
	dbPingAttempts   int
	logger           mlog.LoggerIFace
	NewMutexFn       MutexFactory
	servicesAPI      servicesAPI
	isBinaryParam    bool
	schemaName       string
	configFn         func() *mmModel.Config
}

// MutexFactory is used by the store in plugin mode to generate
// a cluster mutex.
type MutexFactory func(name string) (*cluster.Mutex, error)

// New creates a new SQL implementation of the store.
func New(params Params) (*SQLStore, error) {
	if err := params.CheckValid(); err != nil {
		return nil, err
	}

	params.Logger.Info("connectDatabase", mlog.String("dbType", params.DBType))
	store := &SQLStore{
		// TODO: add replica DB support too.
		db:               params.DB,
		dbType:           params.DBType,
		dbPingAttempts:   params.DBPingAttempts,
		tablePrefix:      params.TablePrefix,
		connectionString: params.ConnectionString,
		logger:           params.Logger,
		NewMutexFn:       params.NewMutexFn,
		servicesAPI:      params.ServicesAPI,
		configFn:         params.ConfigFn,
	}

	var err error
	store.isBinaryParam, err = store.computeBinaryParam()
	if err != nil {
		params.Logger.Error(`Cannot compute binary parameter`, mlog.Err(err))
		return nil, err
	}

	store.schemaName, err = store.GetSchemaName()
	if err != nil {
		params.Logger.Error(`Cannot get schema name`, mlog.Err(err))
		return nil, err
	}

	if !params.SkipMigrations {
		if mErr := store.Migrate(); mErr != nil {
			params.Logger.Error(`Table creation / migration failed`, mlog.Err(mErr))

			return nil, mErr
		}
	}
	return store, nil
}

func (s *SQLStore) IsMariaDB() bool {
	if s.dbType != model.MysqlDBType {
		return false
	}

	row := s.db.QueryRow("SELECT Version()")

	var version string
	if err := row.Scan(&version); err != nil {
		s.logger.Error("error checking database version", mlog.Err(err))
		return false
	}

	return strings.Contains(strings.ToLower(version), "mariadb")
}

// computeBinaryParam returns whether the data source uses binary_parameters
// when using Postgres.
func (s *SQLStore) computeBinaryParam() (bool, error) {
	if s.dbType != model.PostgresDBType {
		return false, nil
	}

	url, err := url.Parse(s.connectionString)
	if err != nil {
		return false, err
	}
	return url.Query().Get("binary_parameters") == "yes", nil
}

// Shutdown close the connection with the store.
func (s *SQLStore) Shutdown() error {
	return s.db.Close()
}

// DBHandle returns the raw sql.DB handle.
// It is used by the mattermostauthlayer to run their own
// raw SQL queries.
func (s *SQLStore) DBHandle() *sql.DB {
	return s.db
}

// DBType returns the DB driver used for the store.
func (s *SQLStore) DBType() string {
	return s.dbType
}

func (s *SQLStore) getQueryBuilder(db sq.BaseRunner) sq.StatementBuilderType {
	builder := sq.StatementBuilder
	if s.dbType == model.PostgresDBType || s.dbType == model.SqliteDBType {
		builder = builder.PlaceholderFormat(sq.Dollar)
	}

	return builder.RunWith(db)
}

func (s *SQLStore) escapeField(fieldName string) string { //nolint:unparam
	if s.dbType == model.MysqlDBType {
		return "`" + fieldName + "`"
	}
	if s.dbType == model.PostgresDBType || s.dbType == model.SqliteDBType {
		return "\"" + fieldName + "\""
	}
	return fieldName
}

func (s *SQLStore) DBVersion() string {
	var version string
	var row *sql.Row

	switch s.dbType {
	case model.MysqlDBType:
		row = s.db.QueryRow("SELECT VERSION()")
	case model.PostgresDBType:
		row = s.db.QueryRow("SHOW server_version")
	case model.SqliteDBType:
		row = s.db.QueryRow("SELECT sqlite_version()")
	default:
		return ""
	}

	if err := row.Scan(&version); err != nil {
		s.logger.Error("error checking database version", mlog.Err(err))
		return ""
	}

	return version
}
