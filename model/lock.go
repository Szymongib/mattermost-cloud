package model

type Lock struct {
	LockAcquiredBy *string
	LockAcquiredAt int64
}
