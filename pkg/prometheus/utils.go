package prometheus

import (
	"fmt"
	"strconv"
	"time"
)

func ParseTimestamp(timestamp string) (time.Time, error) {
	// Try parsing as RFC3339 first
	if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
		return t, nil
	}

	// Try parsing as Unix timestamp
	if unixTime, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		return time.Unix(unixTime, 0), nil
	}

	return time.Time{}, fmt.Errorf("timestamp must be RFC3339 format or Unix timestamp")
}
