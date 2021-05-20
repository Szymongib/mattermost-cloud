// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"fmt"
	"math/rand"
)

const (
	installationDNSFormat = "e2e-test-%s.dev.cloud.mattermost.com"
)

// GetDNS returns e2e test dns with random suffix.
func GetDNS() string {
	installationDNS := fmt.Sprintf(installationDNSFormat, randStringBytes(4))
	return installationDNS
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
