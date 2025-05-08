package pgembed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	t.Logf("Created temp data directory for test: %s", path)
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
	t.Logf("Successfully created and started PostgreSQL instance with config: %+v", config)

	// Get a connection string to ensure it's working at a basic level
	connStr, err := pg.ConnectionString("postgres")
	if err != nil {
		t.Errorf("ConnectionString() failed: %v", err)
	}
	if connStr == "" {
		t.Error("ConnectionString() returned an empty string")
	}
	t.Logf("Obtained connection string: %s", connStr)

	err = pg.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
	t.Log("Successfully stopped PostgreSQL instance.")

	// Test stopping a stopped instance
	err = pg.Stop()
	if err != nil {
		t.Errorf("Stop() on an already stopped instance failed: %v", err)
	}
	t.Log("Stop() on an already stopped instance behaved as expected (no error).")
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
	t.Log("Instance started for database operations.")

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
	t.Logf("DatabaseExists(%s) before creation: %v", testDbName, exists)

	// 2. Create DB
	err = pg.CreateDatabase(testDbName, testOwner)
	if err != nil {
		t.Fatalf("CreateDatabase(%s, %s) failed: %v", testDbName, testOwner, err)
	}
	t.Logf("CreateDatabase(%s, %s) succeeded.", testDbName, testOwner)

	// 3. Check if DB exists (should be true)
	exists, err = pg.DatabaseExists(testDbName)
	if err != nil {
		t.Errorf("DatabaseExists(%s) after creation failed: %v", testDbName, err)
	}
	if !exists {
		t.Errorf("DatabaseExists(%s) was false after creation", testDbName)
	}
	t.Logf("DatabaseExists(%s) after creation: %v", testDbName, exists)

	// 4. Get connection string for the new database
	connStr, err := pg.ConnectionString(testDbName)
	if err != nil {
		t.Errorf("ConnectionString(%s) failed: %v", testDbName, err)
	}
	if connStr == "" {
		t.Errorf("ConnectionString(%s) returned an empty string", testDbName)
	}
	t.Logf("Obtained connection string for %s: %s", testDbName, connStr)
	expectedSuffix := "/" + testDbName
	if !strings.HasSuffix(connStr, expectedSuffix) {
		t.Errorf("ConnectionString for %s was '%s', expected to end with '%s'", testDbName, connStr, expectedSuffix)
	}

	// 5. Drop DB
	err = pg.DropDatabase(testDbName)
	if err != nil {
		t.Fatalf("DropDatabase(%s) failed: %v", testDbName, err)
	}
	t.Logf("DropDatabase(%s) succeeded.", testDbName)

	// 6. Check if DB exists (should be false again)
	exists, err = pg.DatabaseExists(testDbName)
	if err != nil {
		t.Errorf("DatabaseExists(%s) after drop failed: %v", testDbName, err)
	}
	if exists {
		t.Errorf("DatabaseExists(%s) was true after drop", testDbName)
	}
	t.Logf("DatabaseExists(%s) after drop: %v", testDbName, exists)
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
	t.Logf("New() with empty version correctly returned error: %v", err)
}
