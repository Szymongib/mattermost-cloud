package db_migration

//func Test_Stupid(t *testing.T) {
//	t.SkipNow()
//
//	rand.Seed(time.Now().UnixNano())
//	t.Parallel()
//	config, err := readConfig()
//	require.NoError(t, err)
//
//
//	client := model.NewClient(config.CloudURL)
//
//	params := workflow.InstallationFlowParams{
//		DBType:        config.InstallationDBType,
//		FileStoreType: config.InstallationFileStoreType,
//	}
//
//	installationFlow := workflow.NewInstallationFlow(params, client, logrus.WithField("installation-flow", "stupid"))
//
//	steps := []*workflow.Step{
//		{
//			Name:      "CreateInstallation",
//			Func:      installationFlow.CreateInstallation,
//			DependsOn: []string{},
//		},
//		{
//			Name:      "GetCI",
//			Func:      installationFlow.GetCI,
//			DependsOn: []string{"CreateInstallation"},
//		},
//		{
//			Name:      "PopulateSampleData",
//			Func:      installationFlow.PopulateSampleData,
//			DependsOn: []string{"GetCI"},
//		},
//		{
//			Name:      "GetConnectionStrAndExport",
//			Func:      installationFlow.GetConnectionStrAndExport,
//			DependsOn: []string{"PopulateSampleData"},
//		},
//		//{
//		//	Name:      "HibernateInstallation",
//		//	Func:      installationFlow.HibernateInstallation,
//		//	DependsOn: []string{"GetConnectionStrAndExport"},
//		//},
//		//{
//		//	Name:      "WakeUpInstallation",
//		//	Func:      installationFlow.WakeUpInstallation,
//		//	DependsOn: []string{"HibernateInstallation"},
//		//},
//	}
//
//	work := workflow.NewWorkflow(steps)
//
//	err = workflow.RunWorkflow(work, logrus.New())
//	require.NoError(t, err)
//
//	export, err := pkg.GetBulkExportStats(client, installationFlow.Meta.ClusterInstallationID, installationFlow.Meta.InstallationID, logrus.New())
//	require.NoError(t, err)
//
//	logrus.Infof("Export stats: %v. Old stats: %v", export, installationFlow.Meta.BulkExportStats)
//
//
//
//}
