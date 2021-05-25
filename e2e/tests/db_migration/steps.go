// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package db_migration

import (
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
)

func baseMigrationSteps(dbMigFlow *workflow.DBMigrationFlow) []*workflow.Step {
	return []*workflow.Step{
		{
			Name:      "CreateInstallation",
			Func:      dbMigFlow.CreateInstallation,
			DependsOn: []string{},
		},
		{
			Name:      "GetMultiTenantDBID",
			Func:      dbMigFlow.GetMultiTenantDBID,
			DependsOn: []string{"CreateInstallation"},
		},
		{
			Name:      "GetCI",
			Func:      dbMigFlow.GetCI,
			DependsOn: []string{"CreateInstallation"},
		},
		{
			Name:      "PopulateSampleData",
			Func:      dbMigFlow.PopulateSampleData,
			DependsOn: []string{"GetCI"},
		},
		{
			Name:      "GetConnectionStrAndExport",
			Func:      dbMigFlow.GetConnectionStrAndExport,
			DependsOn: []string{"PopulateSampleData"},
		},
		{
			Name:      "HibernateInstallationBeforeMigration",
			Func:      dbMigFlow.HibernateInstallation,
			DependsOn: []string{"GetConnectionStrAndExport"},
		},
		{
			Name:      "RunDBMigration",
			Func:      dbMigFlow.RunDBMigration,
			DependsOn: []string{"HibernateInstallationBeforeMigration"},
		},
		{
			Name:      "WakeUpInstallationAfterMigration",
			Func:      dbMigFlow.WakeUpInstallation,
			DependsOn: []string{"RunDBMigration"},
		},
		{
			Name:      "AssertMigrationSuccessful",
			Func:      dbMigFlow.AssertMigrationSuccessful,
			DependsOn: []string{"WakeUpInstallationAfterMigration"},
		},
	}
}

func commitDBMigrationWorkflow(dbMigFlow *workflow.DBMigrationFlow) *workflow.Workflow {
	steps := baseMigrationSteps(dbMigFlow)

	steps = append(steps, &workflow.Step{
		Name:      "CommitMigration",
		Func:      dbMigFlow.CommitMigration,
		DependsOn: []string{"AssertMigrationSuccessful"},
	})

	return workflow.NewWorkflow(steps)
}

func rollbackDBMigrationWorkflow(dbMigFlow *workflow.DBMigrationFlow) *workflow.Workflow {
	steps := baseMigrationSteps(dbMigFlow)

	steps = append(steps, &workflow.Step{
		Name:      "HibernateInstallationBeforeRollback",
		Func:      dbMigFlow.HibernateInstallation,
		DependsOn: []string{"AssertMigrationSuccessful"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "RollbackMigration",
		Func:      dbMigFlow.RollbackMigration,
		DependsOn: []string{"HibernateInstallationBeforeRollback"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "WakeUpInstallationAfterRollback",
		Func:      dbMigFlow.WakeUpInstallation,
		DependsOn: []string{"RollbackMigration"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "AssertRollbackSuccessful",
		Func:      dbMigFlow.AssertRollbackSuccessful,
		DependsOn: []string{"WakeUpInstallationAfterRollback"},
	})

	return workflow.NewWorkflow(steps)
}
