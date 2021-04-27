// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package components

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TODO: improve those tests

func TestErrors(t *testing.T) {

	err := errors.New("test")

	err = ErrWrap(400, err, "test error 2")

	err = errors.Wrap(err, "test error 3")
	err = errors.Wrap(err, "test error 4")
	err = errors.Wrap(err, "test error 5")
	err = errors.Wrap(err, "test error 6")

	status := ErrToStatus(err)
	assert.Equal(t, 400, status)
	fmt.Println(err.Error())
}
