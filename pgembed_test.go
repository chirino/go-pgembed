package pgembed

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Helper function to create a temporary directory for test data
func tempDir(t *testing.T) string {
	// Use a timestamp to make the directory name unique for concurrent tests if ever needed
	// and to make it easier to identify test runs.
	dirName := "pgtest_" + time.Now().Format("20060102_150405_") + t.Name()
	path := filepath.Join(os.TempDir(), dirName)
	if err := os.MkdirAll(path, 0750); err != nil {
		t.Fatalf("Failed to create temp dir %s: %v", path, err)
	}
	return path
}

func TestNewAndStop(t *testing.T) {
	dataDir := tempDir(t)
	defer os.RemoveAll(dataDir)

	config := Config{
		Version:    "16.0.0", // Using a known version, adjust if needed for your setup
		DataDir:    dataDir,
		RuntimeDir: dataDir, // Can often be the same as DataDir for tests
		Port:       0,       // Use a random port
	}

	pg, err := New(config)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if pg == nil {
		t.Fatal("New() returned nil instance")
	}

	// Get a connection string to ensure it's working at a basic level
	connStr, err := pg.ConnectionString("postgres")
	if err != nil {
		t.Errorf("ConnectionString() failed: %v", err)
	}
	if connStr == "" {
		t.Error("ConnectionString() returned an empty string")
	}

	err = pg.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	// Test stopping a stopped instance
	err = pg.Stop()
	if err != nil {
		t.Errorf("Stop() on an already stopped instance failed: %v", err)
	}
}

func TestDatabaseOperations(t *testing.T) {
	dataDir := tempDir(t)
	defer os.RemoveAll(dataDir)

	config := Config{
		Version:    "16.0.0",
		DataDir:    dataDir,
		RuntimeDir: dataDir,
		Port:       0,
	}

	pg, err := New(config)
	if err != nil {
		t.Fatalf("New() failed for database operations: %v", err)
	}
	defer func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("Stop() in defer failed: %v", err)
		}
	}()

	testDbName := "testdb_gopgembedded"
	testOwner := "testowner" // This owner is not used by Rust but checked in Go wrapper

	// 1. Check if DB exists (should be false)
	exists, err := pg.DatabaseExists(testDbName)
	if err != nil {
		t.Errorf("DatabaseExists(%s) before creation failed: %v", testDbName, err)
	}
	if exists {
		t.Errorf("DatabaseExists(%s) was true before creation", testDbName)
	}

	// 2. Create DB
	err = pg.CreateDatabase(testDbName, testOwner)
	if err != nil {
		t.Fatalf("CreateDatabase(%s, %s) failed: %v", testDbName, testOwner, err)
	}

	// 3. Check if DB exists (should be true)
	exists, err = pg.DatabaseExists(testDbName)
	if err != nil {
		t.Errorf("DatabaseExists(%s) after creation failed: %v", testDbName, err)
	}
	if !exists {
		t.Errorf("DatabaseExists(%s) was false after creation", testDbName)
	}

	// 4. Get connection string for the new database
	connStr, err := pg.ConnectionString(testDbName)
	if err != nil {
		t.Errorf("ConnectionString(%s) failed: %v", testDbName, err)
	}
	if connStr == "" {
		t.Errorf("ConnectionString(%s) returned an empty string", testDbName)
	}
	
	// Use sqlx to create a table and insert data
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		t.Fatalf("sqlx.Connect(%s) failed: %v", connStr, err)
	}
	defer db.Close()

	// Create a table
	createTable := `
	CREATE TABLE IF NOT EXISTS test_table (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL
	)	
	`
	_, err = db.Exec(createTable)
	if err != nil {
		t.Fatalf("Exec(%s) failed: %v", createTable, err)
	}

	// Ensure the connection is closed before trying to drop the database
	if err := db.Close(); err != nil {
		t.Fatalf("error closing database connection before drop: %v", err)
	}

	// 5. Drop DB
	err = pg.DropDatabase(testDbName)
	if err != nil {
		t.Fatalf("DropDatabase(%s) failed: %v", testDbName, err)
	}

	// 6. Check if DB exists (should be false again)
	exists, err = pg.DatabaseExists(testDbName)
	if err != nil {
		t.Errorf("DatabaseExists(%s) after drop failed: %v", testDbName, err)
	}
	if exists {
		t.Errorf("DatabaseExists(%s) was true after drop", testDbName)
	}
}

// TestNewWithoutVersion - ensures New returns an error if version is not specified
func TestNewWithoutVersion(t *testing.T) {
	config := Config{
		// Version: "" // Intentionally omitted
		DataDir: tempDir(t),
	}
	defer os.RemoveAll(config.DataDir)

	_, err := New(config)
	if err == nil {
		t.Fatal("New() with empty version did not return an error")
	}
}
