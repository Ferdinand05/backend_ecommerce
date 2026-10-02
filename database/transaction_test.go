package database

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	if err := godotenv.Load("../.env.test"); err != nil {
		t.Fatalf("failed to load .env.test: %v", err)
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	})

	return db
}

func TestGetDBWithoutTx(t *testing.T) {
	db := openTestDB(t)

	if got := GetDB(context.Background(), db); got != db {
		t.Fatal("GetDB() without tx should return the base db")
	}
}

func TestInjectTxAndGetDB(t *testing.T) {
	db := openTestDB(t)

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin tx: %v", tx.Error)
	}
	defer tx.Rollback()

	ctx := InjectTx(context.Background(), tx)

	if got := GetDB(ctx, db); got != tx {
		t.Fatal("GetDB() with injected tx should return the tx")
	}
}
