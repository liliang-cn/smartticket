package main

import (
	"fmt"
	"log"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/company/smartticket/internal/models"
)

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("./data/smartticket.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 创建密码哈希
	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// 创建管理员用户
	user := models.User{
		Email:        "admin@smartticket.local",
		Username:     "admin",
		PasswordHash: string(hashedPassword),
		FirstName:    "System",
		LastName:     "Administrator",
		Role:         "admin",
		IsActive:     true,
	}
	if err := db.Create(&user).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
	} else {
		fmt.Printf("Created user: %s (ID: %d)\n", user.Email, user.ID)
	}

	// 创建测试客户用户
	customerPassword := "customer123"
	customerHashedPassword, err := bcrypt.GenerateFromPassword([]byte(customerPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash customer password:", err)
	}

	customer := models.User{
		Email:        "customer@smartticket.local",
		Username:     "customer",
		PasswordHash: string(customerHashedPassword),
		FirstName:    "Test",
		LastName:     "Customer",
		Role:         "customer",
		IsActive:     true,
	}
	if err := db.Create(&customer).Error; err != nil {
		log.Printf("Failed to create customer user: %v", err)
	} else {
		fmt.Printf("Created customer user: %s (ID: %d)\n", customer.Email, customer.ID)
	}

	fmt.Println("Test data initialization completed!")
	fmt.Printf("Admin user: %s / %s\n", "admin@smartticket.local", "admin123")
	fmt.Printf("Customer user: %s / %s\n", "customer@smartticket.local", "customer123")
}
