package common

import "time"

const DateLayout = "2006-01-02"

func ParseDate(value string) (time.Time, error) {
	return time.Parse(DateLayout, value)
}

func DateOrNow(value string) (time.Time, error) {
	if value == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return ParseDate(value)
}
