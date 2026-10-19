package utils

import (
	"log"
	"time"
)

// ShowInfoLogs writes an informational line. Never pass passwords, tokens,
// token hashes or credential structs.
func ShowInfoLogs(format string, args ...any) {
	log.Printf("[INFO] "+format, args...)
}

// ShowErrorLogs writes an error line with its context. Same rule: no secrets.
func ShowErrorLogs(context string, err error) {
	log.Printf("[ERROR] %s: %v", context, err)
}

// TimeLayout is the single timestamp format stored in SQLite. Every value is
// UTC with milliseconds, so string comparison in SQL matches time order.
const TimeLayout = "2006-01-02T15:04:05.000Z"

// FormatTime renders t in TimeLayout (UTC).
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeLayout)
}
