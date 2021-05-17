// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

func WaitForFunc(timeout time.Duration, interval time.Duration, isReady func() (bool, error)) error {
	done := time.After(timeout)

	for {
		ready, err := isReady()
		if err != nil {
			return errors.Wrap(err, "while checking if condition is ready")
		}

		if ready {
			return nil
		}

		select {
		case <-done:
			return fmt.Errorf("timeout waiting for condition")
		default:
			time.Sleep(interval)
		}
	}
}
