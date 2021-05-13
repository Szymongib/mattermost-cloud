// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
package db_migration

import (
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
)

func baseMigrationSteps(dbMigFlow *workflow.DBMigrationFlow) []*workflow.Step {
	return []*workflow.Step{
		{
			Name:      "GetMultiTenantDBID",
			Func:      dbMigFlow.GetMultiTenantDBID,
			Done:      false,
			DependsOn: []string{},
		},
		{
			Name:      "CreateInstallation",
			Func:      dbMigFlow.CreateInstallation,
			Done:      false,
			DependsOn: []string{"GetMultiTenantDBID"},
		},
		{
			Name:      "GetCI",
			Func:      dbMigFlow.GetCI,
			Done:      false,
			DependsOn: []string{"CreateInstallation"},
		},
		{
			Name:      "PopulateSampleData",
			Func:      dbMigFlow.PopulateSampleData,
			Done:      false,
			DependsOn: []string{"GetCI"},
		},
		{
			Name:      "GetConnectionStrAndExport",
			Func:      dbMigFlow.GetConnectionStrAndExport,
			Done:      false,
			DependsOn: []string{"PopulateSampleData"},
		},
		{
			Name:      "HibernateInstallationBeforeMigration",
			Func:      dbMigFlow.HibernateInstallation,
			Done:      false,
			DependsOn: []string{"GetConnectionStrAndExport"},
		},
		{
			Name:      "RunDBMigration",
			Func:      dbMigFlow.RunDBMigration,
			Done:      false,
			DependsOn: []string{"HibernateInstallationBeforeMigration"},
		},
		{
			Name:      "WakeUpInstallationAfterMigration",
			Func:      dbMigFlow.WakeUpInstallation,
			Done:      false,
			DependsOn: []string{"RunDBMigration"},
		},
		{
			Name:      "AssertMigrationSuccessful",
			Func:      dbMigFlow.AssertMigrationSuccessful,
			Done:      false,
			DependsOn: []string{"WakeUpInstallationAfterMigration"},
		},
	}
}

func commitDBMigrationWorkflow(dbMigFlow *workflow.DBMigrationFlow) *workflow.Workflow {
	steps := baseMigrationSteps(dbMigFlow)

	steps = append(steps, &workflow.Step{
		Name:      "CommitMigration",
		Func:      dbMigFlow.CommitMigration,
		Done:      false,
		DependsOn: []string{"AssertMigrationSuccessful"},
	})

	return workflow.NewWorkflow(steps)
}

func rollbackDBMigrationWorkflow(dbMigFlow *workflow.DBMigrationFlow) *workflow.Workflow {
	steps := baseMigrationSteps(dbMigFlow)

	steps = append(steps, &workflow.Step{
		Name:      "HibernateInstallationBeforeRollback",
		Func:      dbMigFlow.HibernateInstallation,
		Done:      false,
		DependsOn: []string{"AssertMigrationSuccessful"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "RollbackMigration",
		Func:      dbMigFlow.RollbackMigration,
		Done:      false,
		DependsOn: []string{"HibernateInstallationBeforeRollback"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "WakeUpInstallationAfterRollback",
		Func:      dbMigFlow.WakeUpInstallation,
		Done:      false,
		DependsOn: []string{"RollbackMigration"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "AssertRollbackSuccessful",
		Func:      dbMigFlow.AssertRollbackSuccessful,
		Done:      false,
		DependsOn: []string{"WakeUpInstallationAfterRollback"},
	})

	return workflow.NewWorkflow(steps)
}
