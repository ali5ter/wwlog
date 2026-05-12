package tui

import "time"

// locale encapsulates TLD-derived display preferences for dates and units.
type locale struct {
	tld                string
	weightUnitOverride string
}

func newLocale(tld, weightUnitOverride string) locale {
	return locale{tld: tld, weightUnitOverride: weightUnitOverride}
}

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

// weightUnit returns the unit to display, respecting any config override.
func (l locale) weightUnit(fromAPI string) string {
	if l.weightUnitOverride != "" {
		return l.weightUnitOverride
	}
	if fromAPI != "" {
		return fromAPI
	}
	if l.isUS() {
		return "lb"
	}
	return "kg"
}

// displayWeight returns the value and unit for display, converting the
// API-reported value when a weight_unit config override differs from it.
func (l locale) displayWeight(value float64, apiUnit string) (float64, string) {
	unit := l.weightUnit(apiUnit)
	if apiUnit != "" && unit != apiUnit {
		switch {
		case apiUnit == "kg" && unit == "lb":
			return value * 2.20462, unit
		case apiUnit == "lb" && unit == "kg":
			return value / 2.20462, unit
		}
	}
	return value, unit
}
