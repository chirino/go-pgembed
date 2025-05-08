package pgembed

/*
// These CGO flags assume the Rust library has been compiled and is available.
// You need to compile the Rust library first:
//
//     go generate
//

// Common linker flags needed by Rust standard library and dependencies.
// Adjust if your Rust code has other specific system dependencies.
#cgo darwin LDFLAGS: -L./libs/darwin_arm64 -lrust_pg_embedded_lib -ldl -lm -framework Security -framework CoreFoundation -framework SystemConfiguration -llzma -mmacosx-version-min=15.1
#cgo linux LDFLAGS: -L./libs/linux_amd64 -lrust_pg_embedded_lib -ldl -lm -lrt -lpthread -llzma
#cgo windows LDFLAGS: -L./libs/windows_amd64 -lrust_pg_embedded_lib -lws2_32 -luserenv -ladvapi32 -lbcrypt -lntdll -llzma

// C function declarations matching the Rust FFI.
// Using `typedef struct RustEmbeddedPg RustEmbeddedPg;` for the opaque pointer.
#include <stdlib.h> // For C.free
#include <stdbool.h> // For C._Bool (Go bool)

typedef struct RustEmbeddedPg RustEmbeddedPg; // Opaque struct

RustEmbeddedPg* pg_embedded_create_and_start(
    const char* data_dir_str,
    const char* runtime_dir_str,
    unsigned short port,
    const char* password_str
);

bool pg_embedded_stop(RustEmbeddedPg* pg_ptr);

char* pg_embedded_get_connection_string(const RustEmbeddedPg* pg_ptr, const char* db_name_str);

bool pg_embedded_create_database(RustEmbeddedPg* pg_ptr, const char* db_name_str);

bool pg_embedded_drop_database(RustEmbeddedPg* pg_ptr, const char* db_name_str);

bool pg_embedded_database_exists(const RustEmbeddedPg* pg_ptr, const char* db_name_str);

void pg_embedded_free_string(char* s);
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

// EmbeddedPostgres represents an embedded PostgreSQL instance.
type EmbeddedPostgres struct {
	instance *C.RustEmbeddedPg
	config   Config // Store config for reference
}

// Config holds configuration for the embedded PostgreSQL.
type Config struct {
	// Version of PostgreSQL to use (e.g., "16.2.0", "15.6.0", "14.11.0", etc.). Mandatory.
	// Check postgresql-embedded crate for supported versions.
	Version string
	// DataDir is the path to the PostgreSQL data directory.
	// If empty, a temporary directory managed by the Rust library will be used.
	DataDir string
	// RuntimeDir is the path for runtime files (e.g., sockets).
	// If empty, a temporary directory managed by the Rust library will be used.
	RuntimeDir string
	// Port for PostgreSQL to listen on. If 0, a random available port will be chosen.
	Port uint16
	// Password for the default 'postgres' user. If empty, password may not be set or a default used.
	Password string
}

// New initializes, downloads (if necessary), and starts an embedded PostgreSQL instance.
// The first run for a specific PostgreSQL version might take time to download binaries.
// Binaries are cached by `postgresql-embedded` typically in `~/.embed-postgres/`.
func New(config Config) (*EmbeddedPostgres, error) {
	if config.Version == "" {
		return nil, errors.New("PostgreSQL version must be specified in Config")
	}

	// cVersion := C.CString(config.Version)
	// defer C.free(unsafe.Pointer(cVersion))

	var cDataDir *C.char
	if config.DataDir != "" {
		absDataDir, err := filepath.Abs(config.DataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for DataDir: %w", err)
		}
		// Ensure directory exists if specified, as postgresql-embedded might expect it.
		if err := os.MkdirAll(absDataDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create DataDir %s: %w", absDataDir, err)
		}
		cDataDir = C.CString(absDataDir)
		defer C.free(unsafe.Pointer(cDataDir))
	}

	var cRuntimeDir *C.char
	if config.RuntimeDir != "" {
		absRuntimeDir, err := filepath.Abs(config.RuntimeDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for RuntimeDir: %w", err)
		}
		if err := os.MkdirAll(absRuntimeDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create RuntimeDir %s: %w", absRuntimeDir, err)
		}
		cRuntimeDir = C.CString(absRuntimeDir)
		defer C.free(unsafe.Pointer(cRuntimeDir))
	}

	var cPassword *C.char
	if config.Password != "" {
		cPassword = C.CString(config.Password)
		defer C.free(unsafe.Pointer(cPassword))
	}

	cInstance := C.pg_embedded_create_and_start(
		cDataDir,
		cRuntimeDir,
		C.ushort(config.Port),
		cPassword,
	)

	if cInstance == nil {
		return nil, errors.New("failed to create and start embedded PostgreSQL instance. " +
			"Check console for Rust panic messages or logs. " +
			"Ensure PostgreSQL binaries can be downloaded/run (internet may be required for first download of a version). " +
			"Common issues: invalid version, port conflict, disk space, permissions, or timeout during download/setup.")
	}

	pg := &EmbeddedPostgres{instance: cInstance, config: config}
	runtime.SetFinalizer(pg, (*EmbeddedPostgres).Stop) // Ensure Stop is called on GC if not explicitly called.
	return pg, nil
}

// Stop shuts down and cleans up the embedded PostgreSQL instance.
// It's safe to call Stop multiple times.
// This method is also registered as a finalizer for the EmbeddedPostgres struct.
func (pg *EmbeddedPostgres) Stop() error {
	if pg.instance == nil {
		return nil // Already stopped or never started
	}

	// The finalizer might call this, so ensure we don't try to operate on a nil pg.
	// However, the finalizer is called on pg itself, so `pg` won't be nil here.
	// The primary concern is `pg.instance`.

	stopped := C.pg_embedded_stop(pg.instance)
	pg.instance = nil // Mark as stopped regardless of C call result to prevent reuse

	// Remove the finalizer to prevent it from running again
	runtime.SetFinalizer(pg, nil)

	if !bool(stopped) {
		return errors.New("failed to stop embedded PostgreSQL instance, or it was already stopped by Rust drop")
	}

	return nil
}

// ConnectionString returns a libpq-compatible connection string for the given database name.
// If dbName is empty, "postgres" is typically used as the default database.
func (pg *EmbeddedPostgres) ConnectionString(dbName string) (string, error) {
	if pg.instance == nil {
		return "", errors.New("instance is not running or has been stopped")
	}
	if dbName == "" {
		dbName = "postgres" // Default database
	}

	cDbName := C.CString(dbName)
	defer C.free(unsafe.Pointer(cDbName))

	cConnStr := C.pg_embedded_get_connection_string(pg.instance, cDbName)
	if cConnStr == nil {
		return "", errors.New("failed to get connection string (Rust layer returned null)")
	}
	defer C.pg_embedded_free_string(cConnStr)

	return C.GoString(cConnStr) + "?sslmode=disable", nil
}

// CreateDatabase creates a new database in the embedded instance.
// The default owner is 'postgres' if owner string is empty.
func (pg *EmbeddedPostgres) CreateDatabase(dbName string, owner string) error {
	if pg.instance == nil {
		return errors.New("instance is not running or has been stopped")
	}
	if dbName == "" {
		return errors.New("database name cannot be empty")
	}
	if owner == "" {
		owner = "postgres" // Default owner for PostgreSQL
	}

	cDbName := C.CString(dbName)
	defer C.free(unsafe.Pointer(cDbName))
	// cOwner := C.CString(owner)
	// defer C.free(unsafe.Pointer(cOwner))

	if !bool(C.pg_embedded_create_database(pg.instance, cDbName)) {
		return fmt.Errorf("failed to create database '%s' (owner parameter '%s' is no longer used by the Rust layer)", dbName, owner)
	}
	return nil
}

// DropDatabase drops an existing database from the embedded instance.
func (pg *EmbeddedPostgres) DropDatabase(dbName string) error {
	if pg.instance == nil {
		return errors.New("instance is not running or has been stopped")
	}
	if dbName == "" {
		return errors.New("database name cannot be empty")
	}

	cDbName := C.CString(dbName)
	defer C.free(unsafe.Pointer(cDbName))

	if !bool(C.pg_embedded_drop_database(pg.instance, cDbName)) {
		return fmt.Errorf("failed to drop database '%s'", dbName)
	}
	return nil
}

// DatabaseExists checks if a database with the given name exists.
func (pg *EmbeddedPostgres) DatabaseExists(dbName string) (bool, error) {
	if pg.instance == nil {
		return false, errors.New("instance is not running or has been stopped")
	}
	if dbName == "" {
		return false, errors.New("database name cannot be empty")
	}

	cDbName := C.CString(dbName)
	defer C.free(unsafe.Pointer(cDbName))

	exists := C.pg_embedded_database_exists(pg.instance, cDbName)
	return bool(exists), nil
}
