//! Database query utilities and helpers

/// Query builder for complex database queries
pub struct QueryBuilder {
    table: String,
    conditions: Vec<String>,
    orders: Vec<String>,
    limit: Option<i64>,
    offset: Option<i64>,
}

impl QueryBuilder {
    pub fn new(table: &str) -> Self {
        Self {
            table: table.to_string(),
            conditions: Vec::new(),
            orders: Vec::new(),
            limit: None,
            offset: None,
        }
    }

    pub fn where_condition(mut self, condition: &str) -> Self {
        self.conditions.push(condition.to_string());
        self
    }

    pub fn order_by(mut self, column: &str, direction: &str) -> Self {
        self.orders.push(format!("{} {}", column, direction));
        self
    }

    pub fn limit(mut self, limit: i64) -> Self {
        self.limit = Some(limit);
        self
    }

    pub fn offset(mut self, offset: i64) -> Self {
        self.offset = Some(offset);
        self
    }

    pub fn build(&self) -> String {
        let mut query = format!("SELECT * FROM {}", self.table);

        if !self.conditions.is_empty() {
            query.push_str(" WHERE ");
            query.push_str(&self.conditions.join(" AND "));
        }

        if !self.orders.is_empty() {
            query.push_str(" ORDER BY ");
            query.push_str(&self.orders.join(", "));
        }

        if let Some(limit) = self.limit {
            query.push_str(&format!(" LIMIT {}", limit));
        }

        if let Some(offset) = self.offset {
            query.push_str(&format!(" OFFSET {}", offset));
        }

        query
    }
}

/// Common query helpers
pub struct QueryHelper;

impl QueryHelper {
    /// Build tenant filter condition
    pub fn tenant_filter(tenant_id: &str) -> String {
        format!("tenant_id = '{}'", tenant_id)
    }

    /// Build user permission filter
    pub fn user_permission_filter(user_id: &str, role: &str) -> String {
        match role {
            "SuperAdmin" => "1=1".to_string(),
            _ => format!(
                "(created_by_id = '{}' OR assigned_to_id = '{}' OR contact_id = '{}')",
                user_id, user_id, user_id
            ),
        }
    }

    /// Build date range filter
    pub fn date_range_filter(field: &str, start: &str, end: &str) -> String {
        format!("{} BETWEEN '{}' AND '{}'", field, start, end)
    }

    /// Build status filter
    pub fn status_filter(field: &str, statuses: &[&str]) -> String {
        if statuses.is_empty() {
            "1=1".to_string()
        } else {
            let status_list: Vec<String> = statuses.iter().map(|s| format!("'{}'", s)).collect();
            format!("{} IN ({})", field, status_list.join(", "))
        }
    }

    /// Build full-text search condition
    pub fn text_search_condition(search_query: &str) -> String {
        format!("to_tsvector('english', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', '{}')", search_query)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_query_builder() {
        let query = QueryBuilder::new("tickets")
            .where_condition("status = 'open'")
            .order_by("created_at", "DESC")
            .limit(10)
            .build();

        assert_eq!(
            query,
            "SELECT * FROM tickets WHERE status = 'open' ORDER BY created_at DESC LIMIT 10"
        );
    }

    #[test]
    fn test_query_helpers() {
        assert_eq!(
            QueryHelper::tenant_filter("tenant-123"),
            "tenant_id = 'tenant-123'"
        );

        assert_eq!(
            QueryHelper::user_permission_filter("user-456", "CustomerUser"),
            "(created_by_id = 'user-456' OR assigned_to_id = 'user-456' OR contact_id = 'user-456')"
        );

        assert_eq!(
            QueryHelper::status_filter("status", &["open", "in_progress"]),
            "status IN ('open', 'in_progress')"
        );
    }
}
