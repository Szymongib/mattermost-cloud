// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
package workflow

import "github.com/mattermost/mattermost-cloud/model"

type ProvisionerFlow struct {
	client *model.Client

	Params ProvisionerParams
}

type ProvisionerParams struct {
	ProvisionerURL  string
}


