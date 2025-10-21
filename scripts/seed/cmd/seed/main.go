package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
	"github.com/company/smartticket/internal/seed"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var (
	configFile = flag.String("config", "", "Configuration file path")
	dbPath     = flag.String("db", "", "Database file path (overrides config)")
	outputFile = flag.String("output", "", "Output JSON file for seed data")
	loadFile   = flag.String("load", "", "Load seed data from JSON file")
	force      = flag.Bool("force", false, "Force seeding even if data exists")
	clear      = flag.Bool("clear", false, "Clear all data before seeding")
	verbose    = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("SmartTicket Database Seeder\n\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("  # Generate seed data to file\n")
		fmt.Printf("  %s -output seed_data.json\n\n", os.Args[0])
		fmt.Printf("  # Seed database with generated data\n")
		fmt.Printf("  %s -config configs/config.dev.yaml\n\n", os.Args[0])
		fmt.Printf("  # Seed database from file\n")
		fmt.Printf("  %s -config configs/config.dev.yaml -load seed_data.json\n\n", os.Args[0])
		fmt.Printf("  # Clear and reseed database\n")
		fmt.Printf("  %s -config configs/config.dev.yaml -clear -force\n\n", os.Args[0])
	}

	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if *outputFile != "" {
		// Generate and save seed data to file
		if err := generateSeedDataFile(*outputFile); err != nil {
			log.Fatalf("Failed to generate seed data: %v", err)
		}
		fmt.Printf("Seed data generated and saved to %s\n", *outputFile)
		return
	}

	if *configFile == "" {
		// Try to find default config file
		for _, configPath := range []string{
			"configs/config.dev.yaml",
			"configs/config.local.yaml",
			"configs/config.yaml",
		} {
			if _, err := os.Stat(configPath); err == nil {
				*configFile = configPath
				break
			}
		}
		if *configFile == "" {
			log.Fatal("No configuration file found. Please specify -config option")
		}
	}

	// Set config file if specified
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override database path if provided
	if *dbPath != "" {
		cfg.Database.ConnectionURL = *dbPath
	}

	if *verbose {
		log.Printf("Using database: %s", cfg.Database.ConnectionURL)
	}

	// Initialize database
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Check if database is healthy
	if !db.IsHealthy() {
		log.Fatal("Database is not healthy")
	}

	// Get underlying GORM DB instance
	gormDB := db.GetDB()

	// Auto-migrate database schema before seeding
	if *verbose {
		log.Println("Running database migrations...")
	}

	// Import models package for auto-migration
	// Note: We need to add the actual models here when available
	// For now, we'll create basic tables using the seed structs
	if err := autoMigrateSeedModels(gormDB); err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	if *verbose {
		log.Println("Database migrations completed successfully")
	}

	if *clear {
		// Clear all data
		if err := clearDatabase(gormDB); err != nil {
			log.Fatalf("Failed to clear database: %v", err)
		}
		fmt.Println("Database cleared successfully")
	}

	// Check if data already exists
	if !*force && !*clear {
		if hasData, err := databaseHasData(gormDB); err != nil {
			log.Fatalf("Failed to check database data: %v", err)
		} else if hasData {
			log.Fatal("Database already contains data. Use -force to overwrite or -clear to reseed")
		}
	}

	// Load or generate seed data
	var seedData interface{}
	if *loadFile != "" {
		// Load from file
		seedData, err = loadSeedDataFromFile(*loadFile)
		if err != nil {
			log.Fatalf("Failed to load seed data from file: %v", err)
		}
		fmt.Printf("Loaded seed data from %s\n", *loadFile)
	} else {
		// Generate new seed data
		seedData = generateSeedData()
		if *verbose {
			fmt.Println("Generated new seed data")
		}
	}

	// Seed the database
	if err := seedDatabase(gormDB, seedData); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	fmt.Println("Database seeded successfully!")
}

func generateSeedDataFile(filename string) error {
	data := seed.GenerateSeedData()
	return seed.SaveSeedData(data, filename)
}

func loadSeedDataFromFile(filename string) (interface{}, error) {
	return seed.LoadSeedData(filename)
}

func generateSeedData() interface{} {
	return seed.GenerateSeedData()
}

func seedDatabase(db interface{}, data interface{}) error {
	// Type assertion to get the correct database type
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		return fmt.Errorf("invalid database type")
	}

	seedData, ok := data.(*seed.SeedData)
	if !ok {
		return fmt.Errorf("invalid seed data type")
	}

	return seed.SeedDatabase(gormDB, seedData)
}

func clearDatabase(db *gorm.DB) error {
	log.Println("Clearing database...")

	// Clear tables in reverse order of foreign key dependencies
	tables := []string{
		"llm_providers",
		"knowledge_articles",
		"tickets",
		"settings",
		"users",
		"ticket_statuses",
		"ticket_categories",
		"tenants",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			log.Printf("Warning: Failed to clear table %s: %v", table, err)
		} else {
			log.Printf("Cleared table: %s", table)
		}
	}

	// Reset auto-increment sequences (for SQLite)
	if err := db.Exec("DELETE FROM sqlite_sequence WHERE name IN ('tenants', 'users', 'tickets', 'knowledge_articles', 'llm_providers', 'settings')").Error; err != nil {
		log.Printf("Warning: Failed to reset sequences: %v", err)
	}

	return nil
}

func databaseHasData(db *gorm.DB) (bool, error) {
	// Check if any table has data
	var count int64

	// Check tenants table
	if err := db.Raw("SELECT COUNT(*) FROM tenants").Scan(&count).Error; err != nil {
		// Table might not exist, create it
		return false, nil
	}

	return count > 0, nil
}

// autoMigrateSeedModels auto-migrates all seed data models.
func autoMigrateSeedModels(db *gorm.DB) error {
	// Auto-migrate all seed data models
	models := []interface{}{
		&seed.Tenant{},
		&seed.User{},
		&seed.Ticket{},
		&seed.KnowledgeArticle{},
		&seed.Setting{},
		&seed.LLMProvider{},
		&seed.TicketCategory{},
		&seed.TicketStatus{},
	}

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	return nil
}
