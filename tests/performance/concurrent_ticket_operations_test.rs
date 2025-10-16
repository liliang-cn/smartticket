use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Semaphore;
use tests::common::{TestContext, fixtures};
use smartticket_shared_database::models::*;
use smartticket_shared_auth::{TenantContext, UserData};
use uuid::Uuid;
use sqlx::Row;

/// Benchmark concurrent ticket creation operations
#[tokio::test]
async fn benchmark_concurrent_ticket_creation() -> Result<(), Box<dyn std::error::Error>> {
    let mut context = TestContext::new().await?;

    // Setup test data
    let tenant_id = Uuid::new_v4();
    let admin_user_id = Uuid::new_v4();

    // Create tenant and admin user
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant_id,
        "Performance Test Tenant",
        "perf.test.com",
        SubscriptionTier::Enterprise,
        1000i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    sqlx::query!(
        "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
         VALUES ($1, $2, $3, $4, $5, $6, $7)",
        admin_user_id,
        tenant_id,
        "admin@perf.test.com",
        "admin",
        "Performance Admin",
        "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
        UserRole::TenantAdmin as _
    )
    .execute(context.db.pool())
    .await?;

    // Set application context
    let admin_context = TenantContext::new(
        tenant_id,
        admin_user_id,
        "TenantAdmin".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(admin_context.tenant_id)
        .bind(admin_context.user_id)
        .execute(context.db.pool())
        .await?;

    // Performance test parameters
    let concurrent_workers = 50;
    let tickets_per_worker = 20;
    let total_tickets = concurrent_workers * tickets_per_worker;

    println!("🚀 Starting concurrent ticket creation benchmark");
    println!("   Workers: {}", concurrent_workers);
    println!("   Tickets per worker: {}", tickets_per_worker);
    println!("   Total tickets: {}", total_tickets);

    // Create semaphore to limit concurrent connections
    let semaphore = Arc::new(Semaphore::new(concurrent_workers));
    let db = Arc::new(context.db.clone());
    let tenant_id = Arc::new(tenant_id);

    let start_time = Instant::now();

    // Spawn concurrent tasks
    let mut tasks = Vec::new();

    for worker_id in 0..concurrent_workers {
        let sem = semaphore.clone();
        let db_clone = db.clone();
        let tenant_clone = tenant_id.clone();
        let admin_ctx = admin_context.clone();

        let task = tokio::spawn(async move {
            let _permit = sem.acquire().await.unwrap();

            let worker_start = Instant::now();
            let mut worker_tickets = Vec::new();

            for i in 0..tickets_per_worker {
                let ticket_id = Uuid::new_v4();

                sqlx::query!(
                    "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
                     VALUES ($1, $2, $3, $4, $5, $6)
                     RETURNING ticket_number",
                    ticket_id,
                    *tenant_clone,
                    format!("Performance Test Ticket {}-{}", worker_id, i),
                    format!("This is a performance test ticket from worker {} number {}", worker_id, i),
                    admin_ctx.user_id,
                    admin_ctx.user_id
                )
                .fetch_one(db_clone.pool())
                .await
                .expect("Failed to create ticket");

                worker_tickets.push(ticket_id);
            }

            let worker_duration = worker_start.elapsed();
            (worker_id, worker_tickets.len(), worker_duration)
        });

        tasks.push(task);
    }

    // Wait for all tasks to complete
    let results = futures::future::join_all(tasks).await;
    let total_duration = start_time.elapsed();

    // Calculate statistics
    let mut total_created = 0;
    let mut worker_durations = Vec::new();

    for result in results {
        let (worker_id, tickets_created, duration) = result.unwrap();
        total_created += tickets_created;
        worker_durations.push(duration);
        println!("   Worker {}: {} tickets in {:?}", worker_id, tickets_created, duration);
    }

    // Performance metrics
    let tickets_per_second = total_created as f64 / total_duration.as_secs_f64();
    let avg_worker_duration = worker_durations.iter().sum::<Duration>() / worker_durations.len() as u32;
    let fastest_worker = worker_durations.iter().min().unwrap();
    let slowest_worker = worker_durations.iter().max().unwrap();

    println!("\n📊 Performance Results:");
    println!("   Total tickets created: {}", total_created);
    println!("   Total duration: {:?}", total_duration);
    println!("   Tickets per second: {:.2}", tickets_per_second);
    println!("   Average worker duration: {:?}", avg_worker_duration);
    println!("   Fastest worker: {:?}", fastest_worker);
    println!("   Slowest worker: {:?}", slowest_worker);

    // Verify all tickets were created
    let actual_count: i64 = sqlx::query("SELECT COUNT(*) FROM tickets")
        .fetch_one(context.db.pool())
        .await?
        .get("count");

    assert_eq!(actual_count, total_created as i64, "All tickets should be created");

    // Performance assertions
    assert!(tickets_per_second > 10.0, "Should create at least 10 tickets per second");
    assert!(total_duration < Duration::from_secs(30), "Should complete within 30 seconds");

    println!("✅ Performance benchmark passed!");

    context.cleanup().await?;
    Ok(())
}

/// Benchmark concurrent read operations
#[tokio::test]
async fn benchmark_concurrent_ticket_reads() -> Result<(), Box<dyn std::error::Error>> {
    let mut context = TestContext::new().await?;

    // Setup test data
    let tenant_id = Uuid::new_v4();
    let admin_user_id = Uuid::new_v4();

    // Create tenant and admin user
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant_id,
        "Read Performance Test",
        "readperf.test.com",
        SubscriptionTier::Enterprise,
        1000i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    sqlx::query!(
        "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
         VALUES ($1, $2, $3, $4, $5, $6, $7)",
        admin_user_id,
        tenant_id,
        "admin@readperf.test.com",
        "admin",
        "Read Performance Admin",
        "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
        UserRole::TenantAdmin as _
    )
    .execute(context.db.pool())
    .await?;

    // Set application context
    let admin_context = TenantContext::new(
        tenant_id,
        admin_user_id,
        "TenantAdmin".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(admin_context.tenant_id)
        .bind(admin_context.user_id)
        .execute(context.db.pool())
        .await?;

    // Create test tickets
    let ticket_count = 1000;
    println!("📖 Creating {} test tickets for read benchmark...", ticket_count);

    let mut ticket_ids = Vec::new();

    for i in 0..ticket_count {
        let ticket_id = Uuid::new_v4();

        sqlx::query!(
            "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
             VALUES ($1, $2, $3, $4, $5, $6)",
            ticket_id,
            tenant_id,
            format!("Read Test Ticket {}", i),
            format!("This is ticket number {} for read performance testing", i),
            admin_user_id,
            admin_user_id
        )
        .execute(context.db.pool())
        .await?;

        ticket_ids.push(ticket_id);
    }

    println!("✅ Created {} test tickets", ticket_count);

    // Performance test parameters
    let concurrent_readers = 20;
    let reads_per_reader = ticket_count / concurrent_readers;

    println!("🚀 Starting concurrent read benchmark");
    println!("   Readers: {}", concurrent_readers);
    println!("   Reads per reader: {}", reads_per_reader);
    println!("   Total reads: {}", ticket_count);

    let semaphore = Arc::new(Semaphore::new(concurrent_readers));
    let db = Arc::new(context.db.clone());
    let ticket_ids = Arc::new(ticket_ids);

    let start_time = Instant::now();

    // Spawn concurrent read tasks
    let mut tasks = Vec::new();

    for reader_id in 0..concurrent_readers {
        let sem = semaphore.clone();
        let db_clone = db.clone();
        let tickets_clone = ticket_ids.clone();
        let admin_ctx = admin_context.clone();

        let start_idx = reader_id * reads_per_reader;
        let end_idx = if reader_id == concurrent_readers - 1 {
            ticket_ids.len()
        } else {
            start_idx + reads_per_reader
        };

        let task = tokio::spawn(async move {
            let _permit = sem.acquire().await.unwrap();

            let reader_start = Instant::now();
            let mut successful_reads = 0;

            for i in start_idx..end_idx {
                let ticket_id = tickets_clone[i];

                // Set context for each read operation
                sqlx::query("SELECT set_app_context($1, $2)")
                    .bind(admin_ctx.tenant_id)
                    .bind(admin_ctx.user_id)
                    .execute(db_clone.pool())
                    .await
                    .expect("Failed to set context");

                let ticket = sqlx::query!(
                    "SELECT id, title, ticket_number, created_at FROM tickets WHERE id = $1",
                    ticket_id
                )
                .fetch_optional(db_clone.pool())
                .await
                .expect("Failed to fetch ticket");

                if ticket.is_some() {
                    successful_reads += 1;
                }
            }

            let reader_duration = reader_start.elapsed();
            (reader_id, successful_reads, reader_duration)
        });

        tasks.push(task);
    }

    // Wait for all tasks to complete
    let results = futures::future::join_all(tasks).await;
    let total_duration = start_time.elapsed();

    // Calculate statistics
    let mut total_reads = 0;
    let mut reader_durations = Vec::new();

    for result in results {
        let (reader_id, reads_completed, duration) = result.unwrap();
        total_reads += reads_completed;
        reader_durations.push(duration);
        println!("   Reader {}: {} reads in {:?}", reader_id, reads_completed, duration);
    }

    // Performance metrics
    let reads_per_second = total_reads as f64 / total_duration.as_secs_f64();
    let avg_reader_duration = reader_durations.iter().sum::<Duration>() / reader_durations.len() as u32;
    let fastest_reader = reader_durations.iter().min().unwrap();
    let slowest_reader = reader_durations.iter().max().unwrap();

    println!("\n📊 Read Performance Results:");
    println!("   Total reads: {}", total_reads);
    println!("   Total duration: {:?}", total_duration);
    println!("   Reads per second: {:.2}", reads_per_second);
    println!("   Average reader duration: {:?}", avg_reader_duration);
    println!("   Fastest reader: {:?}", fastest_reader);
    println!("   Slowest reader: {:?}", slowest_reader);

    // Performance assertions
    assert!(reads_per_second > 100.0, "Should achieve at least 100 reads per second");
    assert!(total_duration < Duration::from_secs(10), "Should complete within 10 seconds");

    println!("✅ Read performance benchmark passed!");

    context.cleanup().await?;
    Ok(())
}

/// Benchmark full-text search performance
#[tokio::test]
async fn benchmark_full_text_search() -> Result<(), Box<dyn std::error::Error>> {
    let mut context = TestContext::new().await?;

    // Setup test data
    let tenant_id = Uuid::new_v4();
    let admin_user_id = Uuid::new_v4();

    // Create tenant and admin user
    sqlx::query!(
        "INSERT INTO tenants (id, name, domain, subscription_tier, max_users, data_residency_region)
         VALUES ($1, $2, $3, $4, $5, $6)",
        tenant_id,
        "Search Performance Test",
        "searchperf.test.com",
        SubscriptionTier::Enterprise,
        1000i32,
        "EU"
    )
    .execute(context.db.pool())
    .await?;

    sqlx::query!(
        "INSERT INTO users (id, tenant_id, email, username, full_name, password_hash, role)
         VALUES ($1, $2, $3, $4, $5, $6, $7)",
        admin_user_id,
        tenant_id,
        "admin@searchperf.test.com",
        "admin",
        "Search Performance Admin",
        "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsCm1HLKS",
        UserRole::TenantAdmin as _
    )
    .execute(context.db.pool())
    .await?;

    // Set application context
    let admin_context = TenantContext::new(
        tenant_id,
        admin_user_id,
        "TenantAdmin".to_string(),
    );

    sqlx::query("SELECT set_app_context($1, $2)")
        .bind(admin_context.tenant_id)
        .bind(admin_context.user_id)
        .execute(context.db.pool())
        .await?;

    // Create test tickets with varied content for search testing
    let ticket_count = 500;
    let search_terms = vec!["login", "password", "error", "network", "database", "api", "server", "performance"];

    println!("🔍 Creating {} test tickets for search benchmark...", ticket_count);

    for i in 0..ticket_count {
        let search_term = search_terms[i % search_terms.len()];
        let title = format!("Issue with {} - Ticket {}", search_term, i);
        let description = format!(
            "Customer reported {} issue. The problem occurs when trying to access the {} service. \
             Error message indicates {} connection problem. Need to investigate {} configuration \
             and check {} logs for more details.",
            search_term, search_term, search_term, search_term, search_term
        );

        sqlx::query!(
            "INSERT INTO tickets (id, tenant_id, title, description, contact_id, created_by_id)
             VALUES ($1, $2, $3, $4, $5, $6)",
            Uuid::new_v4(),
            tenant_id,
            title,
            description,
            admin_user_id,
            admin_user_id
        )
        .execute(context.db.pool())
        .await?;
    }

    println!("✅ Created {} test tickets", ticket_count);

    // Benchmark search queries
    let search_queries = vec![
        "login",
        "password error",
        "network connection",
        "database performance",
        "api server",
    ];

    println!("🚀 Starting full-text search benchmark");

    for query in &search_queries {
        let start_time = Instant::now();

        let results = sqlx::query!(
            "SELECT id, title, ticket_number,
                    ts_rank(search_vector, plainto_tsquery('english', $1)) as rank
             FROM (
                 SELECT id, title, ticket_number,
                        to_tsvector('english', title || ' ' || description) as search_vector
                 FROM tickets
             ) as tickets_with_search
             WHERE search_vector @@ plainto_tsquery('english', $1)
             ORDER BY rank DESC
             LIMIT 50",
            query
        )
        .fetch_all(context.db.pool())
        .await?;

        let search_duration = start_time.elapsed();

        println!("   Query '{}': {} results in {:?}", query, results.len(), search_duration);

        // Performance assertions
        assert!(search_duration < Duration::from_millis(100), "Search should complete within 100ms");
        assert!(results.len() > 0, "Search should return results");
    }

    println!("✅ Search performance benchmark passed!");

    context.cleanup().await?;
    Ok(())
}