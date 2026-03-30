// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package migrationstests

import (
	"os"
	"strings"
	"testing"

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

// SetupPluginTestHelper sets up a test helper in plugin mode (with Mattermost
// migrations), which makes migration templates use Mattermost core tables
// (e.g. Preferences instead of focalboard_preferences).
func SetupPluginTestHelper(t *testing.T) (*TestHelper, func()) {
	driverName := strings.TrimSpace(os.Getenv("TEST_DATABASE_DRIVERNAME"))
	if driverName == "" {
		t.Skip("Skipping plugin mode test: no database configured")
	}

	return setupTestHelper(t, true)
}

// SetupTestHelper sets up a test helper in standalone mode (without Mattermost
// migrations), which makes migration templates use focalboard_* tables.
func SetupTestHelper(t *testing.T) (*TestHelper, func()) {
	driverName := strings.TrimSpace(os.Getenv("TEST_DATABASE_DRIVERNAME"))
	if driverName == "" {
		t.Skip("Skipping standalone mode test: no database configured")
	}

	return setupTestHelper(t, false)
}

func setupTestHelper(t *testing.T, withMattermostMigrations bool) (*TestHelper, func()) {
	f := foundation.New(t, NewBoardsMigrator(withMattermostMigrations))

	th := &TestHelper{
		t: t,
		f: f,
	}

	tearDown := func() {
		th.f.TearDown()
	}

	return th, tearDown
}
