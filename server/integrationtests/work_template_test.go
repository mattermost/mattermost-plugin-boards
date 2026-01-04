// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package integrationtests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// This test is there to guarantee that the board templates needed for
// the work template are present in the default templates.
// If this fails, you might need to sync with the channels team.
func TestGetTemplatesForWorkTemplate(t *testing.T) {
	// map[name]trackingTemplateId (SHA256 hashes)
	knownInWorkTemplates := map[string]string{
		"Company Goals & OKRs":   "7bd8aabce55f508a52954cc539aa6ab3654d48e093f183ba5a2aea12216a5712",
		"Competitive Analysis":   "cc298948acc22c99120d6051dc06a468fd5877077ea6b4b712e0eaad387575eb",
		"Content Calendar":       "1710c11b033156671179a32e5fc2824a71bdea818415e434dd47d2da0217c101",
		"Meeting Agenda ":        "b03b2e5b9be6ded96b9c6d6d59923963bd9054894c9829d1b37054efa6084de9",
		"Personal Goals ":        "3fd512dd973ecc8f8b3ffe066ea8409f040a4dfb57d1a75bec96ccb18ee0c135",
		"Personal Tasls ":        "6a619fe3943d47ddbd0681b8c546230f7819dd86636264d34057bfb563787cee",
		"Project Tasks ":         "da397037e9ad7e84d99783ae4fe01f752c1564191bf7bb464a93fb7624a803e7",
		"Roadmap ":               "73bbe8436fac26b1361735078a1d3172bac669b696677fbb31c9f7a12690fe3f",
		"Sales Pipeline CRM":     "e0675727acb40007c37a22ccc72f1f4b8a3a7992614fb1014a04593384cd8c82",
		"Sprint Planner ":        "a80cc39d52b3f355df62629135a0f45d59654eaeae83f3112191659e159c6726",
		"Team Retrospective":     "ca4a7cf46d979b1c0326cb4e9fc8b6b32707c0226bc3be5d1275b2076836a9da",
		"User Research Sessions": "8e1be4728c34efd671387645cf45a5ad0c44a97e4101ec7fb69ee59f8c496a60",
	}
	th := SetupTestHelperPluginMode(t)
	defer th.TearDown()

	clients := setupClients(th)
	th.Client = clients.TeamMember

	err := th.Server.App().InitTemplates()
	require.NoError(t, err, "InitTemplates should not fail")

	rBoards, resp := th.Client.GetTemplatesForTeam("0")
	th.CheckOK(resp)
	require.NotNil(t, rBoards)

	trackingTemplateIDs := []string{}
	for _, board := range rBoards {
		property, _ := board.GetPropertyString("trackingTemplateId")
		if property != "" {
			trackingTemplateIDs = append(trackingTemplateIDs, property)
		}
	}

	// make sure all known templates are in trackingTemplateIds
	for name, ttID := range knownInWorkTemplates {
		found := false
		for _, trackingTemplateID := range trackingTemplateIDs {
			if trackingTemplateID == ttID {
				found = true
				break
			}
		}
		require.True(t, found, "trackingTemplateId %s for %s not found", ttID, name)
	}
}
