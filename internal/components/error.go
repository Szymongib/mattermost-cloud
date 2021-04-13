package components

import (
	"github.com/pkg/errors"
	"net/http"
)

type ErrWithStatus struct {
	err error
	status int
}

func (e *ErrWithStatus) Error() string {
	return e.err.Error()
}

func NewErr(status int, err error) error {
	return &ErrWithStatus{
		err:    err,
		status: status,
	}
}

func ErrWrap(status int, err error, message string) error {
	if err == nil {
		return nil
	}
	return &ErrWithStatus{
		err:    errors.Wrap(err, message),
		status: status,
	}
}

func ErrWrapf(status int, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &ErrWithStatus{
		err:    errors.Wrapf(err, format, args...),
		status: status,
	}
}

func ErrToStatus(err error) int {
	statusErr := &ErrWithStatus{}
	if errors.As(err, &statusErr) {
		return statusErr.status
	}
	return http.StatusInternalServerError
}

