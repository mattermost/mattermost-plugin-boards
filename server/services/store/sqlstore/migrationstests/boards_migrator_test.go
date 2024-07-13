package migrationstests

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/mattermost/morph"
	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/drivers/mysql"
	"github.com/mattermost/morph/drivers/postgres"
	"github.com/mattermost/morph/drivers/sqlite"
	embedded "github.com/mattermost/morph/sources/embedded"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	mmSqlStore "github.com/mattermost/mattermost/server/public/utils/sql"
	"github.com/mattermost/mattermost/server/v8/channels/db"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/store/sqlstore"
)

var tablePrefix = "focalboard_"

type BoardsMigrator struct {
	withMattermostMigrations bool
	connString               string
	driverName               string
	db                       *sql.DB
	store                    *sqlstore.SQLStore
	morphEngine              *morph.Morph
	morphDriver              drivers.Driver
}

func NewBoardsMigrator(withMattermostMigrations bool) *BoardsMigrator {
	return &BoardsMigrator{
		withMattermostMigrations: withMattermostMigrations,
	}
}

func (bm *BoardsMigrator) runMattermostMigrations() error {
	assets := db.Assets()
	assetsList, err := assets.ReadDir(filepath.Join("migrations", bm.driverName))
	if err != nil {
		return err
	}

	assetNames := make([]string, len(assetsList))
	for i, entry := range assetsList {
		assetNames[i] = entry.Name()
	}

	src, err := embedded.WithInstance(&embedded.AssetSource{
		Names: assetNames,
		AssetFunc: func(name string) ([]byte, error) {
			return assets.ReadFile(filepath.Join("migrations", bm.driverName, name))
		},
	})
	if err != nil {
		return err
	}

	driver, err := bm.getDriver()
	if err != nil {
		return err
	}

	options := []morph.EngineOption{
		morph.SetStatementTimeoutInSeconds(1000000),
	}

	engine, err := morph.New(context.Background(), driver, src, options...)
	if err != nil {
		return err
	}
	defer engine.Close()

	return engine.ApplyAll()
}

func (bm *BoardsMigrator) getDriver() (drivers.Driver, error) {
	var driver drivers.Driver
	var err error
	switch bm.driverName {
	case model.PostgresDBType:
		driver, err = postgres.WithInstance(bm.db)
		if err != nil {
			return nil, err
		}
	case model.MysqlDBType:
		driver, err = mysql.WithInstance(bm.db)
		if err != nil {
			return nil, err
		}
	case model.SqliteDBType:
		driver, err = sqlite.WithInstance(bm.db)
		if err != nil {
			return nil, err
		}
	}

	return driver, nil
}

func (bm *BoardsMigrator) getMorphConnection() (*morph.Morph, drivers.Driver, error) {
	driver, err := bm.getDriver()
	if err != nil {
		return nil, nil, err
	}

	assetsList, err := sqlstore.Assets.ReadDir("migrations")
	if err != nil {
		return nil, nil, err
	}
	assetNamesForDriver := make([]string, len(assetsList))
	for i, dirEntry := range assetsList {
		assetNamesForDriver[i] = dirEntry.Name()
	}

	params := map[string]interface{}{
		"prefix":     tablePrefix,
		"postgres":   bm.driverName == model.PostgresDBType,
		"sqlite":     bm.driverName == model.SqliteDBType,
		"mysql":      bm.driverName == model.MysqlDBType,
		"plugin":     bm.withMattermostMigrations,
		"singleUser": false,
	}

	migrationAssets := &embedded.AssetSource{
		Names: assetNamesForDriver,
		AssetFunc: func(name string) ([]byte, error) {
			asset, mErr := sqlstore.Assets.ReadFile("migrations/" + name)
			if mErr != nil {
				return nil, mErr
			}

			tmpl, pErr := template.New("sql").Funcs(bm.store.GetTemplateHelperFuncs()).Parse(string(asset))
			if pErr != nil {
				return nil, pErr
			}
			buffer := bytes.NewBufferString("")

			err = tmpl.Execute(buffer, params)
			if err != nil {
				return nil, err
			}

			return buffer.Bytes(), nil
		},
	}

	src, err := embedded.WithInstance(migrationAssets)
	if err != nil {
		return nil, nil, err
	}

	engine, err := morph.New(context.Background(), driver, src, morph.SetMigrationTableName(fmt.Sprintf("%sschema_migrations", tablePrefix)))
	if err != nil {
		return nil, nil, err
	}

	return engine, driver, nil
}

func (bm *BoardsMigrator) Setup() error {
	var err error
	bm.driverName, bm.connString, err = sqlstore.PrepareNewTestDatabase()
	if err != nil {
		return err
	}

	if bm.driverName == model.MysqlDBType {
		bm.connString, err = mmSqlStore.ResetReadTimeout(bm.connString)
		if err != nil {
			return err
		}

		bm.connString, err = mmSqlStore.AppendMultipleStatementsFlag(bm.connString)
		if err != nil {
			return err
		}
	}

	var dbErr error
	bm.db, dbErr = sql.Open(bm.driverName, bm.connString)
	if dbErr != nil {
		return dbErr
	}

	if newErr := bm.db.Ping(); newErr != nil {
		return newErr
	}

	if bm.withMattermostMigrations {
		if newErr := bm.runMattermostMigrations(); newErr != nil {
			return newErr
		}
	}

	logger, _ := mlog.NewLogger()

	storeParams := sqlstore.Params{
		DBType:           bm.driverName,
		DBPingAttempts:   5,
		ConnectionString: bm.connString,
		TablePrefix:      tablePrefix,
		Logger:           logger,
		DB:               bm.db,
		NewMutexFn: func(name string) (*cluster.Mutex, error) {
			return nil, fmt.Errorf("not implemented")
		},
		SkipMigrations: true,
	}
	bm.store, err = sqlstore.New(storeParams)
	if err != nil {
		return err
	}

	morphEngine, morphDriver, err := bm.getMorphConnection()
	if err != nil {
		return err
	}
	bm.morphEngine = morphEngine
	bm.morphDriver = morphDriver

	return nil
}

func (bm *BoardsMigrator) MigrateToStep(step int) error {
	applied, err := bm.morphDriver.AppliedMigrations()
	if err != nil {
		return err
	}
	currentVersion := len(applied)

	if _, err := bm.morphEngine.Apply(step - currentVersion); err != nil {
		return err
	}

	return nil
}

func (bm *BoardsMigrator) Interceptors() map[int]func() error {
	return map[int]func() error{
		18: bm.store.RunDeletedMembershipBoardsMigration,
		35: func() error {
			return bm.store.RunDeDuplicateCategoryBoardsMigration(35)
		},
	}
}

func (bm *BoardsMigrator) TearDown() error {
	if err := bm.morphEngine.Close(); err != nil {
		return err
	}

	if err := bm.db.Close(); err != nil {
		return err
	}

	return nil
}

func (bm *BoardsMigrator) DriverName() string {
	return bm.driverName
}

func (bm *BoardsMigrator) DB() *sql.DB {
	return bm.db
}
