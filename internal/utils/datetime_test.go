package utils

import (
	"time"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDateTime_Parse(t *testing.T) {
	testCases := []struct {
		layout   string
		input    string
		expected string
		hasError bool
	}{
		{"2006-01-02", "2023-12-25", "2023-12-25", false},
		{"02/01/2006", "25/12/2023", "25/12/2023", false},
		{"2006-01-02", "invalid", "", true},
		{"2006-01-02", "2023-13-01", "", true}, // Invalid month
	}

	for _, tc := range testCases {
		dt, err := ParseDateTime(tc.layout, tc.input)

		if tc.hasError {
			assert.Error(t, err)
			assert.True(t, dt.IsZero())
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, dt.Format(tc.layout))
		}
	}
}

func TestDateTime_Now(t *testing.T) {
	now := Now()
	assert.False(t, now.IsZero())

	// Should be within 1 second of current time
	timeDiff := time.Since(now.Time)
	assert.True(t, timeDiff < time.Second)
}

func TestDateTime_IsBusinessDay(t *testing.T) {
	testCases := []struct {
		date     string
		expected bool
	}{
		{"2023-12-25", true},  // Monday - business day
		{"2023-12-26", true},  // Tuesday - business day
		{"2024-01-01", true},  // Monday - business day
		{"2024-01-02", true},  // Tuesday - business day
		{"2024-01-06", false}, // Saturday - not business day
		{"2024-01-07", false}, // Sunday - not business day
	}

	for _, tc := range testCases {
		dt, _ := ParseDateTime("2006-01-02", tc.date)
		result := dt.IsBusinessDay()
		assert.Equal(t, tc.expected, result, "IsBusinessDay should work correctly for %s", tc.date)
	}
}

func TestDateTime_AddBusinessDays(t *testing.T) {
	// Test adding business days (simplified test, assumes no holidays)
	startDate, _ := ParseDateTime("2006-01-02", "2024-01-01") // Monday

	// Add 1 business day
	result := startDate.AddBusinessDays(1)
	assert.Equal(t, "2024-01-02", result.Format("2006-01-02"))

	// Add 5 business days (should land on next Monday)
	result = startDate.AddBusinessDays(5)
	assert.Equal(t, "2024-01-08", result.Format("2006-01-02"))

	// Add 0 business days
	result = startDate.AddBusinessDays(0)
	assert.Equal(t, startDate.Format("2006-01-02"), result.Format("2006-01-02"))

	// Add negative business days
	result = startDate.AddBusinessDays(-1)
	assert.Equal(t, "2023-12-29", result.Format("2006-01-02"))
}

func TestDateTime_StartOfDay(t *testing.T) {
	input, _ := time.Parse("2006-01-02 15:04:05", "2023-12-25 15:30:45")
	dt := DateTime{Time: input}

	start := dt.StartOfDay()
	expected, _ := time.Parse("2006-01-02 15:04:05", "2023-12-25 00:00:00")
	assert.Equal(t, expected, start.Time)
}

func TestDateTime_EndOfDay(t *testing.T) {
	input, _ := time.Parse("2006-01-02 15:04:05", "2023-12-25 15:30:45")
	dt := DateTime{Time: input}

	end := dt.EndOfDay()
	expected := time.Date(2023, 12, 25, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, expected, end.Time)
}

func TestDateTime_StartOfWeek(t *testing.T) {
	// Test with a Wednesday
	input, _ := time.Parse("2006-01-02", "2023-12-27") // Wednesday
	dt := DateTime{Time: input}

	start := dt.StartOfWeek()
	expected, _ := time.Parse("2006-01-02", "2023-12-25") // Monday
	assert.Equal(t, expected, start.Time)
}

func TestDateTime_EndOfWeek(t *testing.T) {
	// Test with a Wednesday
	input, _ := time.Parse("2006-01-02", "2023-12-27") // Wednesday
	dt := DateTime{Time: input}

	end := dt.EndOfWeek()
	expected := time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC) // Sunday
	assert.Equal(t, expected, end.Time)
}

func TestDateTime_StartOfMonth(t *testing.T) {
	input, _ := time.Parse("2006-01-02", "2023-12-25")
	dt := DateTime{Time: input}

	start := dt.StartOfMonth()
	expected, _ := time.Parse("2006-01-02", "2023-12-01")
	assert.Equal(t, expected, start.Time)
}

func TestDateTime_EndOfMonth(t *testing.T) {
	input, _ := time.Parse("2006-01-02", "2023-12-25")
	dt := DateTime{Time: input}

	end := dt.EndOfMonth()
	expected := time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, expected, end.Time)

	// Test February in leap year
	input, _ = time.Parse("2006-01-02", "2024-02-15")
	dt = DateTime{Time: input}

	end = dt.EndOfMonth()
	expected = time.Date(2024, 2, 29, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, expected, end.Time)
}

func TestDateTime_Age(t *testing.T) {
	// Test age calculation - using current time for realistic age
	birthDate, _ := ParseDateTime("2006-01-02", "1990-01-01")

	age := birthDate.Age()
	assert.True(t, age >= 33 && age <= 35, "Age should be reasonable for 1990 birth year")

	// Test with birthday later in the year
	birthDate, _ = ParseDateTime("2006-01-02", "1990-06-15")
	age = birthDate.Age()
	assert.True(t, age >= 33 && age <= 35, "Age should be reasonable for 1990 birth year")
}

func TestDateTime_DiffInDays(t *testing.T) {
	date1, _ := ParseDateTime("2006-01-02", "2023-12-25")
	date2, _ := ParseDateTime("2006-01-02", "2023-12-30")

	days := date1.DiffInDays(date2)
	assert.Equal(t, -5, days) // date1 - date2 = -5

	// Test reverse order
	days = date2.DiffInDays(date1)
	assert.Equal(t, 5, days) // date2 - date1 = 5

	// Test same day
	days = date1.DiffInDays(date1)
	assert.Equal(t, 0, days)
}

func TestDateTime_DiffInHours(t *testing.T) {
	date1, _ := ParseDateTime("2006-01-02 15:04:05", "2023-12-25 10:00:00")
	date2, _ := ParseDateTime("2006-01-02 15:04:05", "2023-12-25 15:30:00")

	hours := date1.DiffInHours(date2)
	assert.Equal(t, -5, hours) // 10:00 - 15:30 = -5.5 hours, rounded to -5
}

func TestDateTime_FormatHuman(t *testing.T) {
	now := DateTime{Time: time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC)}

	// Test basic human formatting
	result := now.FormatHuman()
	assert.Equal(t, "December 25, 2023 at 12:00 PM", result)
}

func TestDateRange(t *testing.T) {
	start, _ := ParseDateTime("2006-01-02", "2023-12-25")
	end, _ := ParseDateTime("2006-01-02", "2023-12-30")

	range_ := NewDateRange(start, end)
	assert.Equal(t, 5, range_.DaysInRange())
	assert.True(t, range_.Contains(start))
	assert.True(t, range_.Contains(end))
	assert.False(t, range_.Contains(end.Add(0, 0, 1)))
}

func TestDateRange_Overlaps(t *testing.T) {
	range1 := DateRange{
		Start: DateTime{Time: time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)},
		End:   DateTime{Time: time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC)},
	}

	// Overlapping range
	range2 := DateRange{
		Start: DateTime{Time: time.Date(2023, 12, 28, 0, 0, 0, 0, time.UTC)},
		End:   DateTime{Time: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
	}
	assert.True(t, range1.Overlaps(range2))

	// Non-overlapping range
	range3 := DateRange{
		Start: DateTime{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		End:   DateTime{Time: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)},
	}
	assert.False(t, range1.Overlaps(range3))

	// Adjacent ranges (end == start)
	range4 := DateRange{
		Start: DateTime{Time: time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC)},
		End:   DateTime{Time: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)},
	}
	assert.True(t, range1.Overlaps(range4))
}

func TestDateTime_HelperMethods(t *testing.T) {
	// Test current date helpers
	assert.NotEmpty(t, CurrentDate())
	assert.NotEmpty(t, CurrentTime())
	assert.NotEmpty(t, CurrentDateTime())
	assert.NotEmpty(t, CurrentISO())
	assert.NotEmpty(t, CurrentHuman())

	// Test parsing common formats
	date, err := ParseDate("2023-12-25")
	assert.NoError(t, err)
	assert.Equal(t, "2023-12-25", date.FormatDate())

	time_, err := ParseTime("15:30:45")
	assert.NoError(t, err)
	assert.Equal(t, "15:30:45", time_.FormatTime())

	datetime, err := ParseDateTimeStr("2023-12-25 15:30:45")
	assert.NoError(t, err)
	assert.Equal(t, "2023-12-25 15:30:45", datetime.Format(DateTimeFormat))

	iso, err := ParseISO("2023-12-25T15:30:45Z")
	assert.NoError(t, err)
	assert.Equal(t, "2023-12-25T15:30:45Z", iso.FormatISO())
}
