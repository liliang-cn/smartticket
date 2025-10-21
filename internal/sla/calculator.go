package sla

import (
	"errors"
	"time"

	"github.com/company/smartticket/internal/models"
	"gorm.io/gorm"
)

// Calculator provides SLA calculation functionality.
type Calculator struct {
	db *gorm.DB
}

// NewCalculator creates a new SLA calculator instance.
func NewCalculator(db *gorm.DB) *Calculator {
	return &Calculator{db: db}
}

// SLADueDate represents calculated SLA due dates.
type SLADueDate struct {
	ResponseDueDate   time.Time
	ResolutionDueDate time.Time
	ResponseMinutes   int
	ResolutionMinutes int
	BusinessOnly      bool
}

// CalculateSLADueDates calculates due dates for a ticket based on SLA rules.
func (c *Calculator) CalculateSLADueDates(tenantID uint, priority, severity string, productID, serviceID *uint) (*SLADueDate, error) {
	// Find matching SLA rule
	var rule models.SLARule
	query := c.db.Where("tenant_id = ? AND priority = ? AND severity = ? AND is_active = ?",
		tenantID, priority, severity, true)

	// Filter by product if provided
	if productID != nil {
		query = query.Where("(product_id = ? OR product_id IS NULL)", *productID)
	}

	// Filter by service if provided
	if serviceID != nil {
		query = query.Where("(service_id = ? OR service_id IS NULL)", *serviceID)
	}

	// Order by specificity (specific product/service rules first)
	query = query.Order("CASE WHEN product_id IS NOT NULL AND service_id IS NOT NULL THEN 1 " +
		"WHEN product_id IS NOT NULL THEN 2 " +
		"WHEN service_id IS NOT NULL THEN 3 " +
		"ELSE 4 END")

	if err := query.First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default SLA if no rule found
			return c.getDefaultSLADueDates(priority, severity), nil
		}
		return nil, err
	}

	now := time.Now()
	var responseDueDate, resolutionDueDate time.Time

	if rule.BusinessOnly {
		// Calculate business hours due dates
		responseDueDate = c.addBusinessHours(now, rule.ResponseTime)
		resolutionDueDate = c.addBusinessHours(now, rule.ResolutionTime)
	} else {
		// Calculate calendar time due dates
		responseDueDate = now.Add(time.Duration(rule.ResponseTime) * time.Minute)
		resolutionDueDate = now.Add(time.Duration(rule.ResolutionTime) * time.Minute)
	}

	return &SLADueDate{
		ResponseDueDate:   responseDueDate,
		ResolutionDueDate: resolutionDueDate,
		ResponseMinutes:   rule.ResponseTime,
		ResolutionMinutes: rule.ResolutionTime,
		BusinessOnly:      rule.BusinessOnly,
	}, nil
}

// getDefaultSLADueDates returns default SLA due dates when no rule is found.
func (c *Calculator) getDefaultSLADueDates(priority, severity string) *SLADueDate {
	now := time.Now()

	// Default SLA times based on priority
	var responseMinutes, resolutionMinutes int
	switch priority {
	case "critical":
		responseMinutes = 15   // 15 minutes
		resolutionMinutes = 60 // 1 hour
	case "high":
		responseMinutes = 30    // 30 minutes
		resolutionMinutes = 120 // 2 hours
	case "medium":
		responseMinutes = 60    // 1 hour
		resolutionMinutes = 480 // 8 hours
	default: // low
		responseMinutes = 120    // 2 hours
		resolutionMinutes = 1440 // 24 hours
	}

	return &SLADueDate{
		ResponseDueDate:   now.Add(time.Duration(responseMinutes) * time.Minute),
		ResolutionDueDate: now.Add(time.Duration(resolutionMinutes) * time.Minute),
		ResponseMinutes:   responseMinutes,
		ResolutionMinutes: resolutionMinutes,
		BusinessOnly:      false,
	}
}

// addBusinessHours adds business hours to a time.
func (c *Calculator) addBusinessHours(start time.Time, minutes int) time.Time {
	// Simple business hours calculation (Mon-Fri, 9 AM - 6 PM)
	// This is a basic implementation - in production you might want more sophisticated business hour handling
	hours := minutes / 60
	mins := minutes % 60

	result := start
	for i := 0; i < hours; i++ {
		result = result.Add(time.Hour)
		for result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
			// Skip weekends - add days until Monday
			for result.Weekday() != time.Monday {
				result = result.Add(24 * time.Hour)
			}
			// Set to 9 AM start of business day
			result = time.Date(result.Year(), result.Month(), result.Day(), 9, 0, 0, 0, result.Location())
		}

		// Check if we have passed business hours (6 PM)
		if result.Hour() >= 18 {
			// Move to next day 9 AM
			result = result.Add(24 * time.Hour)
			for result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
				result = result.Add(24 * time.Hour)
			}
			result = time.Date(result.Year(), result.Month(), result.Day(), 9, 0, 0, 0, result.Location())
		}
	}

	// Add remaining minutes
	result = result.Add(time.Duration(mins) * time.Minute)

	// Handle overflow past business hours
	if result.Hour() >= 18 {
		// Move to next day 9 AM
		result = result.Add(24 * time.Hour)
		for result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
			result = result.Add(24 * time.Hour)
		}
		result = time.Date(result.Year(), result.Month(), result.Day(), 9, 0, 0, 0, result.Location())
	}

	return result
}

// CheckSLACompliance checks if a ticket meets SLA requirements.
func (c *Calculator) CheckSLACompliance(ticket *models.Ticket) (responseMet, resolutionMet bool, responseMinutes, resolutionMinutes int) {
	if ticket.CreatedAt.IsZero() {
		return true, true, 0, 0
	}

	now := time.Now()
	responseMinutes = int(now.Sub(ticket.CreatedAt).Minutes())

	// For resolution, check if resolved
	resolutionMinutes = 0
	if ticket.Status == "resolved" && ticket.ResolvedAt != nil && !ticket.ResolvedAt.IsZero() {
		resolutionMinutes = int(ticket.ResolvedAt.Sub(ticket.CreatedAt).Minutes())
	}

	// Get SLA due dates
	slaDates, err := c.CalculateSLADueDates(ticket.TenantID, ticket.Priority, ticket.Severity, ticket.ProductID, ticket.ServiceID)
	if err != nil {
		// If no SLA rule found, use default calculation
		defaultSLA := c.getDefaultSLADueDates(ticket.Priority, ticket.Severity)
		responseMet = responseMinutes <= defaultSLA.ResponseMinutes
		if resolutionMinutes > 0 {
			resolutionMet = resolutionMinutes <= defaultSLA.ResolutionMinutes
		} else {
			resolutionMet = true // Not resolved yet
		}
		return responseMet, resolutionMet, responseMinutes, resolutionMinutes
	}

	// Check response SLA
	responseMet = responseMinutes <= slaDates.ResponseMinutes

	// Check resolution SLA
	if resolutionMinutes > 0 {
		resolutionMet = resolutionMinutes <= slaDates.ResolutionMinutes
	} else {
		resolutionMet = true // Not resolved yet
	}

	return responseMet, resolutionMet, responseMinutes, resolutionMinutes
}
