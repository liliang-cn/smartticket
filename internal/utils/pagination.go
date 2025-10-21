package utils

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/company/smartticket/internal/errors"
)

// Pagination represents pagination parameters.
type Pagination struct {
	Page       int   `json:"page" form:"page"`
	PageSize   int   `json:"page_size" form:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// PaginationResult represents paginated result.
type PaginationResult struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Filter represents filtering parameters.
type Filter struct {
	Field    string      `json:"field" form:"field"`
	Operator string      `json:"operator" form:"operator"`
	Value    interface{} `json:"value" form:"value"`
}

// Sort represents sorting parameters.
type Sort struct {
	Field     string `json:"field" form:"field"`
	Direction string `json:"direction" form:"direction"`
}

// QueryBuilder helps build database queries with pagination, filtering, and sorting.
type QueryBuilder struct {
	table       string
	filters     []Filter
	sorts       []Sort
	pagination  *Pagination
	whereClause string
	args        []interface{}
}

// NewQueryBuilder creates a new QueryBuilder.
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:       table,
		filters:     []Filter{},
		sorts:       []Sort{},
		whereClause: "",
		args:        []interface{}{},
	}
}

// AddFilter adds a filter condition.
func (qb *QueryBuilder) AddFilter(field, operator string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, Filter{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return qb
}

// AddFilters adds multiple filter conditions.
func (qb *QueryBuilder) AddFilters(filters []Filter) *QueryBuilder {
	qb.filters = append(qb.filters, filters...)
	return qb
}

// AddSort adds a sort condition.
func (qb *QueryBuilder) AddSort(field, direction string) *QueryBuilder {
	qb.sorts = append(qb.sorts, Sort{
		Field:     field,
		Direction: direction,
	})
	return qb
}

// AddSorts adds multiple sort conditions.
func (qb *QueryBuilder) AddSorts(sorts []Sort) *QueryBuilder {
	qb.sorts = append(qb.sorts, sorts...)
	return qb
}

// SetPagination sets pagination parameters.
func (qb *QueryBuilder) SetPagination(page, pageSize int) *QueryBuilder {
	qb.pagination = &Pagination{
		Page:     page,
		PageSize: pageSize,
	}
	return qb
}

// BuildWhere builds WHERE clause from filters.
func (qb *QueryBuilder) BuildWhere() string {
	if len(qb.filters) == 0 {
		return ""
	}

	var conditions []string
	for _, filter := range qb.filters {
		condition := qb.buildFilterCondition(filter)
		conditions = append(conditions, condition)
	}

	qb.whereClause = strings.Join(conditions, " AND ")
	return qb.whereClause
}

// GetArgs returns arguments for the WHERE clause.
func (qb *QueryBuilder) GetArgs() []interface{} {
	var args []interface{}
	for _, filter := range qb.filters {
		if filter.Value != nil {
			args = append(args, filter.Value)
		}
	}
	return args
}

// BuildOrder builds ORDER BY clause from sorts.
func (qb *QueryBuilder) BuildOrder() string {
	if len(qb.sorts) == 0 {
		return ""
	}

	var orders []string
	for _, sort := range qb.sorts {
		order := fmt.Sprintf("%s %s", sort.Field, strings.ToUpper(sort.Direction))
		orders = append(orders, order)
	}

	return strings.Join(orders, ", ")
}

// buildFilterCondition builds a single filter condition.
func (qb *QueryBuilder) buildFilterCondition(filter Filter) string {
	switch strings.ToUpper(filter.Operator) {
	case "EQ", "=":
		return fmt.Sprintf("%s = ?", qb.quoteIdentifier(filter.Field))
	case "NEQ", "!=":
		return fmt.Sprintf("%s != ?", qb.quoteIdentifier(filter.Field))
	case "GT", ">":
		return fmt.Sprintf("%s > ?", qb.quoteIdentifier(filter.Field))
	case "GTE", ">=":
		return fmt.Sprintf("%s >= ?", qb.quoteIdentifier(filter.Field))
	case "LT", "<":
		return fmt.Sprintf("%s < ?", qb.quoteIdentifier(filter.Field))
	case "LTE", "<=":
		return fmt.Sprintf("%s <= ?", qb.quoteIdentifier(filter.Field))
	case "LIKE":
		return fmt.Sprintf("%s LIKE ?", qb.quoteIdentifier(filter.Field))
	case "ILIKE":
		return fmt.Sprintf("%s ILIKE ?", qb.quoteIdentifier(filter.Field))
	case "IN":
		return fmt.Sprintf("%s IN (?)", qb.quoteIdentifier(filter.Field))
	case "NOT IN":
		return fmt.Sprintf("%s NOT IN (?)", qb.quoteIdentifier(filter.Field))
	case "IS NULL":
		return fmt.Sprintf("%s IS NULL", qb.quoteIdentifier(filter.Field))
	case "IS NOT NULL":
		return fmt.Sprintf("%s IS NOT NULL", qb.quoteIdentifier(filter.Field))
	case "BETWEEN":
		return fmt.Sprintf("%s BETWEEN ? AND ?", qb.quoteIdentifier(filter.Field))
	default:
		return fmt.Sprintf("%s = ?", qb.quoteIdentifier(filter.Field))
	}
}

// quoteIdentifier quotes a database identifier.
func (qb *QueryBuilder) quoteIdentifier(identifier string) string {
	// Simple implementation - in production, use database-specific quoting
	return fmt.Sprintf(`"%s"`, identifier)
}

// ValidatePagination validates pagination parameters.
func ValidatePagination(page, pageSize int, maxPageSize int) (*Pagination, error) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 10
	}

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return &Pagination{
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// CalculatePagination calculates pagination metadata.
func CalculatePagination(page, pageSize int, total int64) Pagination {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// ParseFilters parses filters from URL query parameters.
func ParseFilters(query url.Values, allowedFilters map[string]string) ([]Filter, error) {
	var filters []Filter

	for key, values := range query {
		if !strings.HasPrefix(key, "filter[") || !strings.HasSuffix(key, "]") {
			continue
		}

		// Extract field name from filter[field] format
		field := strings.TrimPrefix(key, "filter[")
		field = strings.TrimSuffix(field, "]")

		// Validate field
		allowed := false
		for allowedField := range allowedFilters {
			if field == allowedField {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, errors.NewValidationError(fmt.Sprintf("Filter field '%s' is not allowed", field))
		}

		// Extract operator and value from filter[field][operator] format
		operatorParts := strings.Split(field, "__")
		fieldName := operatorParts[0]
		operator := "eq" // default operator

		if len(operatorParts) > 1 {
			fieldName = operatorParts[0]
			operator = operatorParts[1]
		}

		// Validate operator
		validOperators := []string{"eq", "neq", "gt", "gte", "lt", "lte", "like", "ilike", "in", "not in", "is_null", "is_not_null", "between"}
		operatorValid := false
		for _, validOp := range validOperators {
			if operator == validOp {
				operatorValid = true
				break
			}
		}
		if !operatorValid {
			return nil, errors.NewValidationError(fmt.Sprintf("Operator '%s' is not allowed", operator))
		}

		// Process values
		for _, value := range values {
			filter := Filter{
				Field:    fieldName,
				Operator: operator,
				Value:    value,
			}
			filters = append(filters, filter)
		}
	}

	return filters, nil
}

// ParseSort parses sorting from URL query parameters.
func ParseSort(query url.Values, allowedSorts map[string]bool) ([]Sort, error) {
	var sorts []Sort

	sortValues, exists := query["sort"]
	if !exists || len(sortValues) == 0 {
		return sorts, nil
	}

	for _, sortStr := range sortValues {
		parts := strings.Split(sortStr, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			sortParts := strings.Split(part, " ")
			if len(sortParts) < 1 || len(sortParts) > 2 {
				return nil, errors.NewValidationError("Sort parameter must be in format 'field' or 'field direction'")
			}

			field := sortParts[0]
			direction := "asc" // default direction

			if len(sortParts) == 2 {
				direction = strings.ToLower(sortParts[1])
			}

			// Validate direction
			if direction != "asc" && direction != "desc" {
				return nil, errors.NewValidationError("Sort direction must be 'asc' or 'desc'")
			}

			// Validate field
			allowed := false
			for allowedField := range allowedSorts {
				if field == allowedField {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, errors.NewValidationError(fmt.Sprintf("Sort field '%s' is not allowed", field))
			}

			sorts = append(sorts, Sort{
				Field:     field,
				Direction: direction,
			})
		}
	}

	return sorts, nil
}

// ParsePaginationFromQuery parses pagination from URL query parameters (renamed to avoid conflict).
func ParsePaginationFromQuery(query url.Values, defaultPageSize, maxPageSize int) (*Pagination, error) {
	pageStr := query.Get("page")
	pageSizeStr := query.Get("page_size")

	page := 1
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return nil, errors.NewValidationError("Page must be a positive integer")
		}
		page = p
	}

	pageSize := defaultPageSize
	if pageSizeStr != "" {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil || ps < 1 {
			return nil, errors.NewValidationError("Page size must be a positive integer")
		}
		pageSize = ps
	}

	return ValidatePagination(page, pageSize, maxPageSize)
}

// BuildLimitOffset builds LIMIT and OFFSET clause for pagination.
func BuildLimitOffset(pagination *Pagination) (string, []interface{}) {
	offset := (pagination.Page - 1) * pagination.PageSize
	return fmt.Sprintf("LIMIT %d OFFSET %d", pagination.PageSize, offset), []interface{}{pagination.PageSize, offset}
}

// CreatePaginationResult creates a pagination result.
func CreatePaginationResult(data interface{}, pagination Pagination) PaginationResult {
	return PaginationResult{
		Data:       data,
		Pagination: pagination,
	}
}

// GetPaginationResponse returns a standard API response with pagination.
func GetPaginationResponse(data interface{}, pagination Pagination) map[string]interface{} {
	return map[string]interface{}{
		"success":    true,
		"data":       data,
		"pagination": pagination,
	}
}

// ParseURLFromQuery parses URL query string into filter, sort, and pagination (renamed to avoid conflict).
func ParseURLFromQuery(queryString string, allowedFilters map[string]string, allowedSorts map[string]bool, defaultPageSize, maxPageSize int) ([]Filter, []Sort, *Pagination, error) {
	query, err := url.ParseQuery(queryString)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse query string: %w", err)
	}

	filters, err := ParseFilters(query, allowedFilters)
	if err != nil {
		return nil, nil, nil, err
	}

	sorts, err := ParseSort(query, allowedSorts)
	if err != nil {
		return nil, nil, nil, err
	}

	pagination, err := ParsePaginationFromQuery(query, defaultPageSize, maxPageSize)
	if err != nil {
		return nil, nil, nil, err
	}

	return filters, sorts, pagination, nil
}

// Default values.
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Filter operators.
const (
	OpEquals       = "eq"
	OpNotEquals    = "neq"
	OpGreaterThan  = "gt"
	OpGreaterEqual = "gte"
	OpLessThan     = "lt"
	OpLessEqual    = "lte"
	OpLike         = "like"
	OpILike        = "ilike"
	OpIn           = "in"
	OpNotIn        = "not in"
	OpIsNull       = "is null"
	OpIsNotNull    = "is not null"
	OpBetween      = "between"
)

// Sort directions.
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)
