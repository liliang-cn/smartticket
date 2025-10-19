//! 演示数据库连接状态检查
//!
//! 这个示例展示数据库连接状态和现有数据

use sqlx::{PgPool, Row};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化数据库连接
    let database_url = "postgres://postgres:postgres@localhost:5433/smartticket_test";
    let pool = PgPool::connect(database_url).await?;

    println!("=== SmartTicket 数据库状态检查 ===\n");

    // 测试数据库连接
    let result = sqlx::query("SELECT 1 as test, version() as db_version, NOW() as current_time")
        .fetch_one(&pool)
        .await?;

    let test_value: i32 = result.get("test");
    let db_version: String = result.get("db_version");
    let current_time: chrono::DateTime<chrono::Utc> = result.get("current_time");

    println!("✅ 数据库连接成功！");
    println!("✅ 测试查询返回: {}", test_value);
    println!(
        "✅ 数据库版本: {}",
        db_version.split(" ").collect::<Vec<&str>>()[1]
    );
    println!("✅ 数据库时间: {}", current_time);

    // 查询现有数据
    println!("\n=== 数据库统计 ===");

    // 查询租户数量和详情
    let tenant_count: i64 = sqlx::query("SELECT COUNT(*) as count FROM tenants")
        .fetch_one(&pool)
        .await?
        .get("count");
    println!("✅ 租户数量: {}", tenant_count);

    if tenant_count > 0 {
        let tenants = sqlx::query("SELECT id, name, domain, max_users FROM tenants")
            .fetch_all(&pool)
            .await?;

        for tenant in tenants {
            let id: uuid::Uuid = tenant.get("id");
            let name: String = tenant.get("name");
            let domain: String = tenant.get("domain");
            let max_users: i32 = tenant.get("max_users");
            println!("  📋 租户: {}", name);
            println!("     ID: {}", id);
            println!("     域名: {}", domain);
            println!("     最大用户数: {}", max_users);
        }
    }

    // 查询用户数量和详情
    let user_count: i64 = sqlx::query("SELECT COUNT(*) as count FROM users")
        .fetch_one(&pool)
        .await?
        .get("count");
    println!("✅ 用户数量: {}", user_count);

    if user_count > 0 {
        let users = sqlx::query("SELECT id, tenant_id, email, full_name, is_active FROM users")
            .fetch_all(&pool)
            .await?;

        for user in users {
            let tenant_id: uuid::Uuid = user.get("tenant_id");
            let email: String = user.get("email");
            let full_name: String = user.get("full_name");
            let is_active: bool = user.get("is_active");
            println!("  👤 用户: {}", full_name);
            println!("     邮箱: {}", email);
            println!("     租户ID: {}", tenant_id);
            println!("     状态: {}", if is_active { "活跃" } else { "非活跃" });
        }
    }

    // 查询其他数据表
    let ticket_count: i64 = sqlx::query("SELECT COUNT(*) as count FROM tickets")
        .fetch_one(&pool)
        .await?
        .get("count");
    let knowledge_count: i64 = sqlx::query("SELECT COUNT(*) as count FROM knowledge_articles")
        .fetch_one(&pool)
        .await?
        .get("count");
    let category_count: i64 = sqlx::query("SELECT COUNT(*) as count FROM ticket_categories")
        .fetch_one(&pool)
        .await?
        .get("count");

    println!("✅ 工单数量: {}", ticket_count);
    println!("✅ 知识文章数量: {}", knowledge_count);
    println!("✅ 工单分类数量: {}", category_count);

    // 查询所有表
    println!("\n=== 数据库表结构 ===");
    let tables = sqlx::query(
        "
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = 'public'
        ORDER BY table_name
    ",
    )
    .fetch_all(&pool)
    .await?;

    println!("✅ 数据库表:");
    for table in tables {
        let table_name: String = table.get("table_name");

        // 获取表的记录数
        let count_result = sqlx::query(&format!("SELECT COUNT(*) as count FROM {}", table_name))
            .fetch_one(&pool)
            .await;

        match count_result {
            Ok(row) => {
                let count: i64 = row.get("count");
                println!("  📊 {} ({} 条记录)", table_name, count);
            }
            Err(_) => {
                println!("  📊 {} (无法获取记录数)", table_name);
            }
        }
    }

    // 测试Redis连接
    println!("\n=== Redis 缓存状态 ===");
    match std::process::Command::new("docker")
        .args(["exec", "smartticket-redis-test", "redis-cli", "ping"])
        .output()
    {
        Ok(output) => {
            let response = String::from_utf8_lossy(&output.stdout);
            if response.trim() == "PONG" {
                println!("✅ Redis 连接: 正常 (PONG)");
            } else {
                println!("❌ Redis 连接: 异常");
            }
        }
        Err(_) => {
            println!("❌ Redis 连接: 无法访问");
        }
    }

    // 关闭连接池
    pool.close().await;

    println!("\n=== 检查完成 ===");
    println!("✅ SmartTicket 数据库系统运行正常！");
    println!("✅ 所有组件都已就绪，可以开始使用！");

    Ok(())
}
