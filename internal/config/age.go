package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DaySeconds int64 = 24 * 60 * 60

func DaysToSeconds(days int) int64 {
	return int64(days) * DaySeconds
}

func SecondsLabel(seconds int64) string {
	if seconds <= 0 {
		return "disabled"
	}
	if seconds%DaySeconds == 0 {
		days := seconds / DaySeconds
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	return fmt.Sprintf("%d hours", seconds/3600)
}

func ParseAgeDuration(raw string) (int64, bool) {
	value := strings.TrimSpace(strings.Trim(raw, `"'`))
	if value == "" {
		return 0, false
	}

	if strings.HasPrefix(value, "P") {
		duration, ok := parseSimpleISO8601Duration(value)
		return duration, ok
	}

	re := regexp.MustCompile(`(?i)^\s*(\d+)\s*([a-z]+)\s*$`)
	match := re.FindStringSubmatch(value)
	if len(match) != 3 {
		return 0, false
	}

	amount, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, false
	}

	switch strings.ToLower(match[2]) {
	case "s", "sec", "second", "seconds":
		return amount, true
	case "m", "min", "minute", "minutes":
		return amount * 60, true
	case "h", "hr", "hour", "hours":
		return amount * 3600, true
	case "d", "day", "days":
		return amount * DaySeconds, true
	case "w", "week", "weeks":
		return amount * 7 * DaySeconds, true
	default:
		return 0, false
	}
}

func ParseRFC3339Cutoff(raw string, now time.Time) (int64, bool) {
	value := strings.Trim(strings.TrimSpace(raw), `"'`)
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0, false
	}
	age := now.Sub(t)
	if age < 0 {
		return 0, true
	}
	return int64(age.Seconds()), true
}

func parseSimpleISO8601Duration(value string) (int64, bool) {
	re := regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)
	match := re.FindStringSubmatch(value)
	if len(match) != 5 {
		return 0, false
	}

	var total int64
	multipliers := []int64{DaySeconds, 3600, 60, 1}
	for i := 1; i < len(match); i++ {
		if match[i] == "" {
			continue
		}
		amount, err := strconv.ParseInt(match[i], 10, 64)
		if err != nil {
			return 0, false
		}
		total += amount * multipliers[i-1]
	}
	return total, true
}
