package tui

import "time"

// locale encapsulates TLD-derived display preferences for dates and units.
type locale struct {
	tld string
}

func newLocale(tld string) locale { return locale{tld: tld} }

func (l locale) isUS() bool { return l.tld == "com" }

// dateShort formats a YYYY-MM-DD date as a short display string.
// US: "Mon, Jan 2"  · others: "Mon, 2 Jan"
func (l locale) dateShort(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	if l.isUS() {
		return t.Format("Mon, Jan 2")
	}
	return t.Format("Mon, 2 Jan")
}

// dateLong formats a YYYY-MM-DD date as a long display string.
// US: "Monday, January 2 2006"  · others: "Monday 2 January 2006"
func (l locale) dateLong(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	if l.isUS() {
		return t.Format("Monday, January 2 2006")
	}
	return t.Format("Monday 2 January 2006")
}

// weightUnit returns the unit string from the API response, falling back to
// a TLD-derived default if the API returns an empty value.
func (l locale) weightUnit(fromAPI string) string {
	if fromAPI != "" {
		return fromAPI
	}
	if l.isUS() {
		return "lb"
	}
	return "kg"
}
