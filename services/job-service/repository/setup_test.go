// Package repository provides shared test setup for all repository tests.
package repository

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/openprint/openprint/internal/testutil"
)

var (
	testDB *testutil.TestDB
	ctx    = context.Background()
)

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("TestMain: Starting test database setup...")

	var err error
	testDB, err = testutil.SetupPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to setup test database: %v", err)
	}

	if testDB == nil {
		log.Fatalf("testDB is nil after SetupPostgresContainer")
	}
	if testDB.Pool == nil {
		log.Fatalf("testDB.Pool is nil after SetupPostgresContainer")
	}

	log.Println("TestMain: Database setup complete, running tests...")
	defer func() {
		log.Println("TestMain: Cleaning up...")
		testutil.Cleanup(testDB)
	}()

	exitCode := m.Run()
	log.Printf("TestMain: Tests finished with exit code: %d", exitCode)
	os.Exit(exitCode)
}

// init is called before TestMain
func init() {
	log.Println("init: testSetup file loaded")
}
