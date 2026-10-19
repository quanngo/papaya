package tenantconfig

import "time"

// AddBusinessDays returns t moved forward by n working days (Monday–Friday),
// keeping the time of day. A claim submitted on a weekend starts counting
// from Monday: Saturday + 1 business day = Monday. Public holidays are not
// considered (documented assumption).
func AddBusinessDays(t time.Time, n int) time.Time {
	d := t
	for added := 0; added < n; {
		d = d.AddDate(0, 0, 1)
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			added++
		}
	}
	return d
}
