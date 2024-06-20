package integrationtests

import (
	"github.com/mattermost/mattermost-plugin-boards/server/services/store"

	mmModel "github.com/mattermost/mattermost/server/public/model"
)

type TestStore struct {
	store.Store
	license *mmModel.License
}

func NewTestEnterpriseStore(store store.Store) *TestStore {
	usersValue := 10000
	trueValue := true
	falseValue := false
	license := &mmModel.License{
		Features: &mmModel.Features{
			Users:                     &usersValue,
			LDAP:                      &trueValue,
			LDAPGroups:                &trueValue,
			MFA:                       &trueValue,
			GoogleOAuth:               &trueValue,
			Office365OAuth:            &trueValue,
			OpenId:                    &trueValue,
			Compliance:                &trueValue,
			Cluster:                   &trueValue,
			Metrics:                   &trueValue,
			MHPNS:                     &trueValue,
			SAML:                      &trueValue,
			Elasticsearch:             &trueValue,
			Announcement:              &trueValue,
			ThemeManagement:           &trueValue,
			EmailNotificationContents: &trueValue,
			DataRetention:             &trueValue,
			MessageExport:             &trueValue,
			CustomPermissionsSchemes:  &trueValue,
			CustomTermsOfService:      &trueValue,
			GuestAccounts:             &trueValue,
			GuestAccountsPermissions:  &trueValue,
			IDLoadedPushNotifications: &trueValue,
			LockTeammateNameDisplay:   &trueValue,
			EnterprisePlugins:         &trueValue,
			AdvancedLogging:           &trueValue,
			Cloud:                     &falseValue,
			SharedChannels:            &trueValue,
			RemoteClusterService:      &trueValue,
			FutureFeatures:            &trueValue,
		},
	}

	testStore := &TestStore{
		Store:   store,
		license: license,
	}

	return testStore
}

func NewTestProfessionalStore(store store.Store) *TestStore {
	usersValue := 10000
	trueValue := true
	falseValue := false
	license := &mmModel.License{
		Features: &mmModel.Features{
			Users:                     &usersValue,
			LDAP:                      &falseValue,
			LDAPGroups:                &falseValue,
			MFA:                       &trueValue,
			GoogleOAuth:               &trueValue,
			Office365OAuth:            &trueValue,
			OpenId:                    &trueValue,
			Compliance:                &falseValue,
			Cluster:                   &falseValue,
			Metrics:                   &trueValue,
			MHPNS:                     &trueValue,
			SAML:                      &trueValue,
			Elasticsearch:             &trueValue,
			Announcement:              &trueValue,
			ThemeManagement:           &trueValue,
			EmailNotificationContents: &trueValue,
			DataRetention:             &trueValue,
			MessageExport:             &trueValue,
			CustomPermissionsSchemes:  &trueValue,
			CustomTermsOfService:      &trueValue,
			GuestAccounts:             &trueValue,
			GuestAccountsPermissions:  &trueValue,
			IDLoadedPushNotifications: &trueValue,
			LockTeammateNameDisplay:   &trueValue,
			EnterprisePlugins:         &falseValue,
			AdvancedLogging:           &trueValue,
			Cloud:                     &falseValue,
			SharedChannels:            &trueValue,
			RemoteClusterService:      &falseValue,
			FutureFeatures:            &trueValue,
		},
	}

	testStore := &TestStore{
		Store:   store,
		license: license,
	}

	return testStore
}

func (s *TestStore) GetLicense() *mmModel.License {
	return s.license
}
