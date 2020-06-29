package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseAppleTimestamp interprets an apple timestamp into a Go time object
func ParseAppleTimestamp(timestamp float64) (time.Time, error) {
	floatParts := strings.Split(fmt.Sprintf("%.3f", timestamp), ".")

	// Timestamp
	timeint, err := strconv.ParseInt(floatParts[0], 10, 64)
	if err != nil {
		return time.Now(), err
	}
	seconds := int64(timeint + 978310800)

	// Timestamp nano
	timenano, err := strconv.ParseInt(floatParts[1], 10, 64)
	if err != nil {
		return time.Now(), err
	}
	nanoseconds := timenano

	return time.Unix(seconds, nanoseconds), nil
}
