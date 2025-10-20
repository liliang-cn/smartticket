package utils

import (
	"time"
)

// Time utilities

// DateTime represents a wrapper around time.Time with helper methods
type DateTime struct {
	time.Time
}

// NewDateTime creates a new DateTime instance
func NewDateTime(t time.Time) DateTime {
	return DateTime{Time: t}
}

// Now returns current time as DateTime
func Now() DateTime {
	return DateTime{Time: time.Now()}
}

// UTC returns current UTC time as DateTime
func UTC() DateTime {
	return DateTime{Time: time.Now().UTC()}
}

// ParseDateTime parses a string into DateTime
func ParseDateTime(layout, value string) (DateTime, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return DateTime{}, err
	}
	return DateTime{Time: t}, nil
}

// ParseDateTimeWithLocation parses a string into DateTime with timezone
func ParseDateTimeWithLocation(layout, value string, loc *time.Location) (DateTime, error) {
	t, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		return DateTime{}, err
	}
	return DateTime{Time: t}, nil
}

// Format formats DateTime to string
func (dt DateTime) Format(layout string) string {
	return dt.Time.Format(layout)
}

// FormatISO formats DateTime to ISO 8601 string
func (dt DateTime) FormatISO() string {
	return dt.Time.Format(time.RFC3339)
}

// FormatHuman formats DateTime to human-readable string
func (dt DateTime) FormatHuman() string {
	return dt.Time.Format("January 2, 2006 at 3:04 PM")
}

// FormatDate formats DateTime to date string
func (dt DateTime) FormatDate() string {
	return dt.Time.Format("2006-01-02")
}

// FormatTime formats DateTime to time string
func (dt DateTime) FormatTime() string {
	return dt.Time.Format("15:04:05")
}

// FormatHumanTime formats DateTime to human-readable time
func (dt DateTime) FormatHumanTime() string {
	return dt.Time.Format("3:04 PM")
}

// StartOfDay returns the beginning of the day
func (dt DateTime) StartOfDay() DateTime {
	year, month, day := dt.Time.Date()
	return DateTime{Time: time.Date(year, month, day, 0, 0, 0, 0, dt.Time.Location())}
}

// EndOfDay returns the end of the day
func (dt DateTime) EndOfDay() DateTime {
	year, month, day := dt.Time.Date()
	return DateTime{Time: time.Date(year, month, day, 23, 59, 59, 999999999, dt.Time.Location())}
}

// StartOfWeek returns the beginning of the week (Monday)
func (dt DateTime) StartOfWeek() DateTime {
	weekday := int(dt.Time.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	start := dt.Time.AddDate(0, 0, -(weekday - 1))
	return DateTime{Time: start}
}

// EndOfWeek returns the end of the week (Sunday)
func (dt DateTime) EndOfWeek() DateTime {
	start := dt.StartOfWeek()
	end := start.Time.AddDate(0, 0, 7).Add(-time.Nanosecond)
	return DateTime{Time: end}
}

// StartOfMonth returns the beginning of the month
func (dt DateTime) StartOfMonth() DateTime {
	year, month, _ := dt.Time.Date()
	return DateTime{Time: time.Date(year, month, 1, 0, 0, 0, 0, dt.Time.Location())}
}

// EndOfMonth returns the end of the month
func (dt DateTime) EndOfMonth() DateTime {
	year, month, _ := dt.Time.Date()
	nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, dt.Time.Location())
	end := nextMonth.Add(-time.Nanosecond)
	return DateTime{Time: end}
}

// Add adds specified duration to DateTime
func (dt DateTime) Add(years, months, days int) DateTime {
	return DateTime{Time: dt.Time.AddDate(years, months, days)}
}

// Sub subtracts specified duration from DateTime
func (dt DateTime) Sub(years, months, days int) DateTime {
	return DateTime{Time: dt.Time.AddDate(-years, -months, -days)}
}

// AddHours adds hours to DateTime
func (dt DateTime) AddHours(hours int) DateTime {
	return DateTime{Time: dt.Time.Add(time.Duration(hours) * time.Hour)}
}

// AddMinutes adds minutes to DateTime
func (dt DateTime) AddMinutes(minutes int) DateTime {
	return DateTime{Time: dt.Time.Add(time.Duration(minutes) * time.Minute)}
}

// AddSeconds adds seconds to DateTime
func (dt DateTime) AddSeconds(seconds int) DateTime {
	return DateTime{Time: dt.Time.Add(time.Duration(seconds) * time.Second)}
}

// Diff calculates difference between two DateTimes
func (dt DateTime) Diff(other DateTime) time.Duration {
	return dt.Time.Sub(other.Time)
}

// DiffInDays calculates difference in days (ignoring time)
func (dt DateTime) DiffInDays(other DateTime) int {
	hours := dt.Diff(other).Hours()
	return int(hours / 24)
}

// DiffInHours calculates difference in hours
func (dt DateTime) DiffInHours(other DateTime) int {
	return int(dt.Diff(other).Hours())
}

// DiffInMinutes calculates difference in minutes
func (dt DateTime) DiffInMinutes(other DateTime) int {
	return int(dt.Diff(other).Minutes())
}

// IsBefore checks if DateTime is before another DateTime
func (dt DateTime) IsBefore(other DateTime) bool {
	return dt.Time.Before(other.Time)
}

// IsAfter checks if DateTime is after another DateTime
func (dt DateTime) IsAfter(other DateTime) bool {
	return dt.Time.After(other.Time)
}

// IsEqual checks if DateTime is equal to another DateTime
func (dt DateTime) IsEqual(other DateTime) bool {
	return dt.Time.Equal(other.Time)
}

// IsZero checks if DateTime is zero time
func (dt DateTime) IsZero() bool {
	return dt.Time.IsZero()
}

// IsToday checks if DateTime is today
func (dt DateTime) IsToday() bool {
	now := time.Now()
	return dt.Year() == now.Year() && dt.Month() == now.Month() && dt.Day() == now.Day()
}

// IsYesterday checks if DateTime is yesterday
func (dt DateTime) IsYesterday() bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return dt.Year() == yesterday.Year() && dt.Month() == yesterday.Month() && dt.Day() == yesterday.Day()
}

// IsTomorrow checks if DateTime is tomorrow
func (dt DateTime) IsTomorrow() bool {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return dt.Year() == tomorrow.Year() && dt.Month() == tomorrow.Month() && dt.Day() == tomorrow.Day()
}

// IsWeekend checks if DateTime is on a weekend
func (dt DateTime) IsWeekend() bool {
	weekday := dt.Time.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// IsWeekday checks if DateTime is on a weekday
func (dt DateTime) IsWeekday() bool {
	return !dt.IsWeekend()
}

// Age calculates age from birth date
func (dt DateTime) Age() int {
	now := time.Now()
	years := now.Year() - dt.Year()
	if now.Month() < dt.Month() || (now.Month() == dt.Month() && now.Day() < dt.Day()) {
		years--
	}
	return years
}

// ToUnix returns Unix timestamp
func (dt DateTime) ToUnix() int64 {
	return dt.Time.Unix()
}

// ToUnixMilli returns Unix timestamp in milliseconds
func (dt DateTime) ToUnixMilli() int64 {
	return dt.Time.UnixMilli()
}

// FromUnix creates DateTime from Unix timestamp
func FromUnix(sec int64) DateTime {
	return DateTime{Time: time.Unix(sec, 0)}
}

// FromUnixMilli creates DateTime from Unix timestamp in milliseconds
func FromUnixMilli(msec int64) DateTime {
	return DateTime{Time: time.Unix(0, msec*int64(time.Millisecond))}
}

// Timezone returns DateTime in specified timezone
func (dt DateTime) Timezone(loc *time.Location) DateTime {
	return DateTime{Time: dt.Time.In(loc)}
}

// UTC returns DateTime in UTC timezone
func (dt DateTime) UTC() DateTime {
	return DateTime{Time: dt.Time.UTC()}
}

// Local returns DateTime in local timezone
func (dt DateTime) Local() DateTime {
	return DateTime{Time: dt.Time.Local()}
}

// Business day calculations

// IsBusinessDay checks if DateTime is a business day (Mon-Fri)
func (dt DateTime) IsBusinessDay() bool {
	weekday := dt.Time.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

// AddBusinessDays adds business days to DateTime
func (dt DateTime) AddBusinessDays(days int) DateTime {
	result := dt.Time

	if days >= 0 {
		added := 0
		for added < days {
			result = result.AddDate(0, 0, 1)
			testDateTime := DateTime{Time: result}
			if testDateTime.IsBusinessDay() {
				added++
			}
		}
	} else {
		subtracted := 0
		for subtracted > days { // days is negative, so we go until we subtract enough
			result = result.AddDate(0, 0, -1)
			testDateTime := DateTime{Time: result}
			if testDateTime.IsBusinessDay() {
				subtracted--
			}
		}
	}

	return DateTime{Time: result}
}

// PreviousBusinessDay returns previous business day
func (dt DateTime) PreviousBusinessDay() DateTime {
	result := dt.Time
	for {
		result = result.AddDate(0, 0, -1)
		testDateTime := DateTime{Time: result}
		if testDateTime.IsBusinessDay() {
			break
		}
	}
	return DateTime{Time: result}
}

// NextBusinessDay returns next business day
func (dt DateTime) NextBusinessDay() DateTime {
	result := dt.Time
	for {
		result = result.AddDate(0, 0, 1)
		testDateTime := DateTime{Time: result}
		if testDateTime.IsBusinessDay() {
			break
		}
	}
	return DateTime{Time: result}
}

// Range utilities

// DateRange represents a date range
type DateRange struct {
	Start DateTime
	End   DateTime
}

// NewDateRange creates a new DateRange
func NewDateRange(start, end DateTime) DateRange {
	return DateRange{Start: start, End: end}
}

// DaysInRange returns number of days in range
func (dr DateRange) DaysInRange() int {
	return int(dr.End.Time.Sub(dr.Start.Time).Hours() / 24)
}

// Contains checks if a DateTime is within the range
func (dr DateRange) Contains(dt DateTime) bool {
	return (dt.Time.Equal(dr.Start.Time) || dt.Time.After(dr.Start.Time)) &&
		(dt.Time.Equal(dr.End.Time) || dt.Time.Before(dr.End.Time))
}

// Overlaps checks if two date ranges overlap
func (dr DateRange) Overlaps(other DateRange) bool {
	return dr.Contains(other.Start) || dr.Contains(other.End) ||
		other.Contains(dr.Start) || other.Contains(dr.End)
}

// Duration utilities

// DurationBetween calculates duration between two DateTimes
func DurationBetween(start, end DateTime) time.Duration {
	return end.Time.Sub(start.Time)
}

// Common date/time formats
const (
	DateFormat     = "2006-01-02"
	TimeFormat     = "15:04:05"
	DateTimeFormat = "2006-01-02 15:04:05"
	ISOFormat      = time.RFC3339
	ISOFormatShort = "2006-01-02T15:04:05Z"
	HumanFormat    = "January 2, 2006 at 3:04 PM"
)

// Parse common formats
func ParseDate(date string) (DateTime, error) {
	return ParseDateTime(DateFormat, date)
}

func ParseTime(timeStr string) (DateTime, error) {
	return ParseDateTime(TimeFormat, timeStr)
}

func ParseDateTimeStr(dateTimeStr string) (DateTime, error) {
	return ParseDateTime(DateTimeFormat, dateTimeStr)
}

func ParseISO(dateTimeStr string) (DateTime, error) {
	return ParseDateTime(ISOFormat, dateTimeStr)
}

// Current time helpers
func CurrentDate() string {
	return Now().FormatDate()
}

func CurrentTime() string {
	return Now().FormatTime()
}

func CurrentDateTime() string {
	return Now().Format(DateTimeFormat)
}

func CurrentISO() string {
	return Now().FormatISO()
}

func CurrentHuman() string {
	return Now().FormatHuman()
}
