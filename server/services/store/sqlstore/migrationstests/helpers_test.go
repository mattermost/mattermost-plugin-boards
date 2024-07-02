package migrationstests

import (
	"os"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mgdelacroix/foundation"
)

type TestHelper struct {
	t *testing.T
	f *foundation.Foundation
}

func (th *TestHelper) IsPostgres() bool {
	return th.f.DB().DriverName() == "postgres"
}

func (th *TestHelper) IsMySQL() bool {
	return th.f.DB().DriverName() == "mysql"
}

func (th *TestHelper) IsSQLite() bool {
	return th.f.DB().DriverName() == "sqlite3"
}

func SetupPluginTestHelper(t *testing.T) (*TestHelper, func()) {
	dbType := strings.TrimSpace(os.Getenv("FOCALBOARD_STORE_TEST_DB_TYPE"))
	if dbType == "" || dbType == model.SqliteDBType {
		t.Skip("Skipping plugin mode test for SQLite")
	}

	return setupTestHelper(t)
}

func SetupTestHelper(t *testing.T) (*TestHelper, func()) {
	return setupTestHelper(t)
}

func setupTestHelper(t *testing.T) (*TestHelper, func()) {
	f := foundation.New(t, NewBoardsMigrator(true))

	th := &TestHelper{
		t: t,
		f: f,
	}

	tearDown := func() {
		th.f.TearDown()
	}

	return th, tearDown
}
