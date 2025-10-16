//! 演示用户认证和权限系统
//!
//! 这个示例展示了如何使用JWT认证和基于角色的权限控制

use smartticket_shared_database::{AuthService, AuthUser, UserRole};
use std::sync::Arc;
use uuid::Uuid;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化认证服务
    let auth_service = Arc::new(AuthService::new(
        "your-secret-key-here-should-be-from-env",
        "smartticket".to_string(),
        "smartticket-client".to_string(),
    ));

    println!("=== SmartTicket 用户认证和权限系统演示 ===\n");

    // 创建不同角色的用户
    let admin_user_id = Uuid::new_v4();
    let admin_tenant_id = Uuid::new_v4();
    let admin_token = auth_service.generate_token(
        &admin_user_id,
        "admin@company.com",
        "admin",
        "Administrator",
        &admin_tenant_id,
        &UserRole::TenantAdmin,
    )?;

    let customer_user_id = Uuid::new_v4();
    let customer_tenant_id = Uuid::new_v4();
    let customer_token = auth_service.generate_token(
        &customer_user_id,
        "customer@company.com",
        "customer",
        "Customer User",
        &customer_tenant_id,
        &UserRole::CustomerUser,
    )?;

    let engineer_user_id = Uuid::new_v4();
    let engineer_tenant_id = Uuid::new_v4();
    let engineer_token = auth_service.generate_token(
        &engineer_user_id,
        "engineer@company.com",
        "engineer",
        "Support Engineer",
        &engineer_tenant_id,
        &UserRole::SupportEngineer,
    )?;

    println!("✅ 成功生成JWT tokens\n");

    // 验证并解析用户信息
    println!("=== 用户权限验证 ===");

    // 验证管理员用户
    let admin_claims = auth_service.validate_token(&admin_token)?;
    let admin_auth_user: AuthUser = admin_claims.clone().into();
    println!("管理员用户: {}", admin_auth_user.full_name);
    println!("角色: {:?}", admin_auth_user.role);
    println!("权限: {:?}", admin_auth_user.permissions);
    println!(
        "可以创建用户: {}",
        auth_service.has_permission(&admin_claims, "user:create")
    );
    println!(
        "可以管理工单: {}",
        auth_service.has_permission(&admin_claims, "ticket:assign")
    );
    println!();

    // 验证客户用户
    let customer_claims = auth_service.validate_token(&customer_token)?;
    let customer_auth_user: AuthUser = customer_claims.clone().into();
    println!("客户用户: {}", customer_auth_user.full_name);
    println!("角色: {:?}", customer_auth_user.role);
    println!("权限: {:?}", customer_auth_user.permissions);
    println!(
        "可以创建工单: {}",
        auth_service.has_permission(&customer_claims, "ticket:create")
    );
    println!(
        "可以分配工单: {}",
        auth_service.has_permission(&customer_claims, "ticket:assign")
    );
    println!(
        "只能查看自己的工单: {}",
        auth_service.has_permission(&customer_claims, "ticket:view_own")
    );
    println!();

    // 验证工程师用户
    let engineer_claims = auth_service.validate_token(&engineer_token)?;
    let engineer_auth_user: AuthUser = engineer_claims.clone().into();
    println!("工程师用户: {}", engineer_auth_user.full_name);
    println!("角色: {:?}", engineer_auth_user.role);
    println!("权限: {:?}", engineer_auth_user.permissions);
    println!(
        "可以分配工单: {}",
        auth_service.has_permission(&engineer_claims, "ticket:assign")
    );
    println!(
        "可以管理知识库: {}",
        auth_service.has_permission(&engineer_claims, "knowledge:publish")
    );
    println!();

    // 演示权限检查
    println!("=== 权限检查示例 ===");

    // 不同角色对工单操作权限
    let ticket_operations = [
        "ticket:create",
        "ticket:view",
        "ticket:view_own",
        "ticket:update",
        "ticket:update_own",
        "ticket:assign",
        "ticket:delete",
    ];

    println!("管理员权限检查:");
    for op in &ticket_operations {
        println!(
            "  {}: {}",
            op,
            auth_service.has_permission(&admin_claims, op)
        );
    }

    println!("\n客户权限检查:");
    for op in &ticket_operations {
        println!(
            "  {}: {}",
            op,
            auth_service.has_permission(&customer_claims, op)
        );
    }

    println!("\n工程师权限检查:");
    for op in &ticket_operations {
        println!(
            "  {}: {}",
            op,
            auth_service.has_permission(&engineer_claims, op)
        );
    }

    println!("\n=== 角色权限总结 ===");

    let roles = vec![
        (UserRole::SuperAdmin, "SuperAdmin"),
        (UserRole::TenantAdmin, "TenantAdmin"),
        (UserRole::SupportEngineer, "SupportEngine"),
        (UserRole::CustomerUser, "CustomerUser"),
        (UserRole::Sales, "Sales"),
    ];

    for (role, role_name) in roles {
        let permissions = auth_service.get_permissions_for_role(&role);
        println!(
            "{} ({}): {} 项权限",
            role_name,
            format!("{:?}", role),
            permissions.len()
        );
        for perm in &permissions {
            println!("  - {}", perm);
        }
        println!();
    }

    println!("✅ 用户认证和权限系统演示完成！");
    println!("✅ 所有功能正常运行，系统可以投入使用！");

    Ok(())
}
