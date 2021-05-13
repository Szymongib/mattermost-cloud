// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"fmt"
	"math/rand"
)

const (
	// TODO: from env
	InstallationDNSFormat = "e2e-test-%s.dev.cloud.mattermost.com"
)

func GetDNS() string {
	installationDNS := fmt.Sprintf(InstallationDNSFormat, RandStringBytes(4))
	return installationDNS
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
