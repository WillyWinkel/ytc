package utils

import (
	"fmt"
	"strings"
	"time"
)

// SplitAndTrim splits a comma-separated string and trims whitespace from each part, omitting empty results.
func SplitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// HumanDuration returns a human-readable duration string for a time.Duration.
func HumanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	m := int(d.Minutes()) % 60
	switch {
	case days > 0 && h > 0 && m > 0:
		return strings.TrimSpace(strings.Join([]string{plural(days, "d"), plural(h, "h"), plural(m, "m")}, " "))
	case days > 0 && h > 0:
		return strings.TrimSpace(strings.Join([]string{plural(days, "d"), plural(h, "h")}, " "))
	case days > 0 && m > 0:
		return strings.TrimSpace(strings.Join([]string{plural(days, "d"), plural(m, "m")}, " "))
	case days > 0:
		return plural(days, "d")
	case h > 0 && m > 0:
		return strings.TrimSpace(strings.Join([]string{plural(h, "h"), plural(m, "m")}, " "))
	case h > 0:
		return plural(h, "h")
	case m > 0:
		return plural(m, "m")
	}
	return "0m"
}

func plural(val int, suffix string) string {
	return fmt.Sprintf("%d%s", val, suffix)
}

// ParseICalTimeToHuman parses an iCal time string and returns the parsed time and a human-readable string.
func ParseICalTimeToHuman(value string) (time.Time, string) {
	layouts := []string{
		"20060102T150405Z",
		"20060102T150405",
		"20060102",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, t.Format("02.01.2006 15:04")
		}
	}
	return time.Time{}, value
}
