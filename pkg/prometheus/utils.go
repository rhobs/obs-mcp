package prometheus

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseTimestamp(timestamp string) (time.Time, error) {
	// Handle NOW keyword (case-insensitive)
	if strings.EqualFold(timestamp, "NOW") {
		return time.Now(), nil
	}

	// Try parsing as RFC3339 first
	if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
		return t, nil
	}

	// Try parsing as Unix timestamp
	if unixTime, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		return time.Unix(unixTime, 0), nil
	}

	return time.Time{}, fmt.Errorf("timestamp must be RFC3339 format, Unix timestamp, or NOW")
}
