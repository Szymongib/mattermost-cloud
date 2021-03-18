package utils

import "time"

func TimeFromMillis(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}
