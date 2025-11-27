package main

import (
	"fmt"
	"log"
	"os"

	"ai-ba/internal/domain/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "postgres://postgres:456123@localhost:5432/aiba?sslmode=disable"
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	var tables []string
	if err := db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables).Error; err != nil {
		log.Printf("Error listing tables: %v", err)
	} else {
		fmt.Println("\n--- TABLES ---")
		for _, t := range tables {
			var count int64
			db.Table(t).Count(&count)
			fmt.Printf("%s (rows: %d)\n", t, count)
		}
	}

	// Check users table columns
	var columns []struct {
		ColumnName string
		DataType   string
	}
	if err := db.Raw("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'users'").Scan(&columns).Error; err != nil {
		log.Printf("Error listing user columns: %v", err)
	} else {
		fmt.Println("\n--- USERS COLUMNS ---")
		for _, c := range columns {
			fmt.Printf("%s (%s)\n", c.ColumnName, c.DataType)
		}
	}

	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		log.Printf("Error finding users: %v", err)
	} else {
		fmt.Println("--- USERS ---")
		for _, u := range users {
			fmt.Printf("ID: %d, Email: %s, Role: '%s', Name: %s\n", u.ID, u.Email, u.Role, u.Name)
		}
	}

	var drafts []models.Draft
	if err := db.Find(&drafts).Error; err != nil {
		log.Printf("Error finding drafts: %v", err)
	} else {
		fmt.Println("\n--- DRAFTS ---")
		for _, d := range drafts {
			fmt.Printf("ID: %d, Title: %s, Status: %s, UserID: %d\n", d.ID, d.Title, d.Status, d.UserID)
		}
	}

	var orphanedCount int64
	if err := db.Model(&models.Message{}).Where("session_id NOT IN (?)", db.Table("sessions").Select("id")).Count(&orphanedCount).Error; err != nil {
		log.Printf("Error counting orphaned messages: %v", err)
	} else {
		fmt.Printf("\n--- ORPHANED MESSAGES ---\nCount: %d\n", orphanedCount)

		if orphanedCount > 0 {
			fmt.Println("Deleting orphaned messages...")
			if err := db.Where("session_id NOT IN (?)", db.Table("sessions").Select("id")).Delete(&models.Message{}).Error; err != nil {
				log.Printf("Error deleting orphans: %v", err)
			} else {
				fmt.Println("Successfully deleted orphaned messages.")
			}
		}
	}
}
