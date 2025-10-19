use uuid::Uuid;
use smartticket_shared_error::SmartTicketError;

/// Assert that an error has the expected error type
pub fn assert_error_type(error: &SmartTicketError, expected_type: &str) {
    match error {
        SmartTicketError::Database(_) => assert_eq!(expected_type, "Database"),
        SmartTicketError::Authentication(_) => assert_eq!(expected_type, "Authentication"),
        SmartTicketError::Authorization(_) => assert_eq!(expected_type, "Authorization"),
        SmartTicketError::Validation(_) => assert_eq!(expected_type, "Validation"),
        SmartTicketError::NotFound(_) => assert_eq!(expected_type, "NotFound"),
        SmartTicketError::Conflict(_) => assert_eq!(expected_type, "Conflict"),
        SmartTicketError::ExternalService(_) => assert_eq!(expected_type, "ExternalService"),
        SmartTicketError::Io(_) => assert_eq!(expected_type, "Io"),
        _ => panic!("Unexpected error type: {:?}", error),
    }
}

/// Assert that tenant isolation is properly enforced
pub fn assert_tenant_isolation(expected_tenant_id: Uuid, actual_tenant_id: Uuid) {
    assert_eq!(
        expected_tenant_id, actual_tenant_id,
        "Tenant isolation violated: expected {}, got {}",
        expected_tenant_id, actual_tenant_id
    );
}

/// Assert that a UUID is valid (not nil)
pub fn assert_valid_uuid(uuid: Uuid) {
    assert_ne!(uuid, Uuid::nil(), "UUID should not be nil");
}

/// Assert that a string is not empty
pub fn assert_not_empty(value: &str, field_name: &str) {
    assert!(!value.is_empty(), "{} should not be empty", field_name);
}

/// Assert that a count is within expected range
pub fn assert_count_in_range(actual: i64, min: i64, max: i64, field_name: &str) {
    assert!(
        actual >= min && actual <= max,
        "{} count {} should be between {} and {}",
        field_name, actual, min, max
    );
}