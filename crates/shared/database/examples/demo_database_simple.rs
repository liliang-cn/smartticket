//! 简化的数据库连接演示
//!
//! 这个示例展示了如何使用默认配置连接到数据库

use smartticket_shared_config::database::{DatabaseConfig, SslMode};
use smartticket_shared_database::DatabaseConnection;
use sqlx::Row;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化数据库连接 - 使用正确的字段结构
    let config = DatabaseConfig {
        host: "localhost".to_string(),
        port: 5434,
        database_name: "smartticket".to_string(),
        username: "postgres".to_string(),
        password: "postgres".to_string(),
        ssl_mode: SslMode::Disable,
        max_connections: 10,
        min_connections: 1,
        connect_timeout: 30,
        idle_timeout: 600,
    };

    let db_connection = DatabaseConnection::new(&config).await?;

    println!("=== SmartTicket 简化数据库连接测试 ===\n");

    // 测试数据库连接 - 修复executor问题
    let result = sqlx::query("SELECT 1 as test")
        .fetch_one(db_connection.pool())
        .await?;

    let test_value: i32 = result.get("test");
    println!("✅ 数据库连接成功！测试查询返回: {}", test_value);

    println!("\n✅ 简化数据库连接测试完成！");

    Ok(())
}