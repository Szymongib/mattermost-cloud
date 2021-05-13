// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package db_migration

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBMigration_Commit(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Parallel()

	test, err := SetupDBMigrationCommitTest()
	require.NoError(t, err)
	//defer func() {
	//	fmt.Println(test.DBMigrationTestData)
	//	err := test.SaveToFile("commit_test.json")
	//	if err != nil {
	//		t.Log("Failed to save commit test to json file: ", err.Error())
	//	}
	//}()

	//webhookTester, err := NewWebhookMigrationTestSuite(params.ProvisionerURL)
	//require.NoError(t, err)
	//
	//err = webhookTester.StartServer()
	//require.NoError(t, err)
	//defer func() {
	//	err = webhookTester.CleanupWebhook()
	//	assert.NoError(t, err)
	//}()
	//
	//// Wait for server
	//time.Sleep(5*time.Second)

	err = test.Run()
	assert.NoError(t, err)

	//err = webhookTester.GetResults()
	//require.NoError(t, err)
}

func TestDBMigration_Rollback(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	t.Parallel()

	test, err := SetupDBMigrationRollbackTest()
	require.NoError(t, err)

	err = test.Run()
	assert.NoError(t, err)
}
