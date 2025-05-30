use postgresql_embedded::blocking::PostgreSQL as BlockingPostgresql;
use postgresql_embedded::Error::DatabaseInitializationError;
use postgresql_embedded::Settings;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::path::PathBuf;
use std::ptr;
use std::time::Duration;

/// Opaque type representing the embedded PostgreSQL instance.
type EmbeddedPg = BlockingPostgresql;

/// Result structure for pg_embedded_create_and_start.
#[repr(C)]
pub struct pgStartResult {
    pg_ptr: *mut EmbeddedPg,
    error_msg: *mut c_char,
}

type PgStartResult = pgStartResult;

/// Helper to convert Rust String to C char pointer.
/// The caller (C/Go) is responsible for freeing this string using `pg_embedded_free_string`.
fn string_to_c_char_ptr(s: String) -> *mut c_char {
    CString::new(s)
        .unwrap_or_else(|_| CString::new("Error: Failed to create CString").unwrap())
        .into_raw()
}

/// Helper to convert C char pointer to Rust String.
/// Returns Ok(String) or Utf8Error. Assumes null-terminated C string.
unsafe fn c_char_ptr_to_string(ptr: *const c_char) -> Result<String, std::str::Utf8Error> {
    if ptr.is_null() {
        return Ok(String::new()); // Or handle as an error if null is not expected
    }
    CStr::from_ptr(ptr).to_str().map(String::from)
}

#[no_mangle]
pub extern "C" fn pg_embedded_create_and_start(
    data_dir_c: *const c_char,
    _runtime_dir_c: *const c_char,
    port: u16,
    password_c: *const c_char,
) -> PgStartResult {
    let mut settings = Settings::default();
    settings.timeout = Some(Duration::from_secs(90)); // Increased timeout for setup/start

    if !data_dir_c.is_null() {
        match unsafe { c_char_ptr_to_string(data_dir_c) } {
            Ok(s) if !s.is_empty() => {
                settings.data_dir = PathBuf::from(s);
            }
            Err(e) => {
                let error_str = format!("failed to convert data_dir_c to string: {}", e);
                return PgStartResult {
                    pg_ptr: ptr::null_mut(),
                    error_msg: string_to_c_char_ptr(error_str),
                };
            }
            _ => {}
        }
    }

    if port > 0 {
        settings.port = port;
    }

    if !password_c.is_null() {
        match unsafe { c_char_ptr_to_string(password_c) } {
            Ok(s) if !s.is_empty() => {
                settings.password = s;
            }
            Err(e) => {
                let error_str = format!("failed to convert password_c to string: {}", e);
                return PgStartResult {
                    pg_ptr: ptr::null_mut(),
                    error_msg: string_to_c_char_ptr(error_str),
                };
            }
            _ => {}
        }
    }

    let mut pg = BlockingPostgresql::new(settings);

    if let Err(e) = pg.setup() {
        let error_str = match e {
            DatabaseInitializationError(reason) => format!("setup failed: {}", reason),
            _ => format!("setup failed: {}", e.to_string()),
        };
        return PgStartResult {
            pg_ptr: ptr::null_mut(),
            error_msg: string_to_c_char_ptr(error_str),
        };
    }

    if let Err(e) = pg.start() {
        let error_str = format!("start failed: {}", e.to_string());
        return PgStartResult {
            pg_ptr: ptr::null_mut(),
            error_msg: string_to_c_char_ptr(error_str),
        };
    }

    PgStartResult {
        pg_ptr: Box::into_raw(Box::new(pg)),
        error_msg: ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn pg_embedded_stop(pg_ptr: *mut EmbeddedPg) -> bool {
    if pg_ptr.is_null() {
        return false;
    }
    // Reconstitute the Box and let it drop, which calls `pg.stop()` if not already stopped
    // and handles cleanup via the Drop trait.
    let pg = unsafe { Box::from_raw(pg_ptr) };
    let result = pg.stop();
    // pg is dropped when it goes out of scope here.
    result.is_ok()
}

#[no_mangle]
pub extern "C" fn pg_embedded_get_connection_string(
    pg_ptr: *const EmbeddedPg,
    db_name_c: *const c_char,
) -> *mut c_char {
    if pg_ptr.is_null() {
        return std::ptr::null_mut();
    }
    let pg = unsafe { &*pg_ptr };
    let db_name =
        unsafe { c_char_ptr_to_string(db_name_c).unwrap_or_else(|_| "postgres".to_string()) };

    let settings = pg.settings();
    let user = if settings.username.is_empty() {
        "postgres".to_string()
    } else {
        settings.username.clone() // Clone to get a String, or we can work with &str
    };
    let host = "localhost"; // postgresql-embedded runs on localhost
    let port = settings.port;

    let userinfo_part = if !settings.password.is_empty() {
        // Note: Passwords with special characters might need URL encoding.
        // This basic construction assumes simple passwords or that the Go driver handles it.
        format!("{}:{}@", user, settings.password)
    } else {
        format!("{}@", user)
    };

    let conn_str = format!(
        "postgresql://{}{}:{}/{}",
        userinfo_part, host, port, db_name
    );
    string_to_c_char_ptr(conn_str)
}

#[no_mangle]
pub extern "C" fn pg_embedded_create_database(
    pg_ptr: *mut EmbeddedPg,
    db_name_c: *const c_char,
) -> bool {
    if pg_ptr.is_null() || db_name_c.is_null() {
        return false;
    }
    let pg = unsafe { &mut *pg_ptr };
    let db_name = match unsafe { c_char_ptr_to_string(db_name_c) } {
        Ok(s) if !s.is_empty() => s,
        _ => return false,
    };

    pg.create_database(&db_name).is_ok()
}

#[no_mangle]
pub extern "C" fn pg_embedded_drop_database(
    pg_ptr: *mut EmbeddedPg,
    db_name_c: *const c_char,
) -> bool {
    if pg_ptr.is_null() || db_name_c.is_null() {
        return false;
    }
    let pg = unsafe { &mut *pg_ptr };
    let db_name = match unsafe { c_char_ptr_to_string(db_name_c) } {
        Ok(s) if !s.is_empty() => s,
        _ => return false,
    };

    pg.drop_database(&db_name).is_ok()
}

#[no_mangle]
pub extern "C" fn pg_embedded_database_exists(
    pg_ptr: *const EmbeddedPg,
    db_name_c: *const c_char,
) -> bool {
    if pg_ptr.is_null() || db_name_c.is_null() {
        return false;
    }
    let pg = unsafe { &*pg_ptr };
    let db_name = match unsafe { c_char_ptr_to_string(db_name_c) } {
        Ok(s) if !s.is_empty() => s,
        _ => return false,
    };

    pg.database_exists(&db_name).unwrap_or(false)
}

/// Frees a string that was allocated by Rust and passed to C.
#[no_mangle]
pub extern "C" fn pg_embedded_free_string(s: *mut c_char) {
    if s.is_null() {
        return;
    }
    unsafe {
        let _ = CString::from_raw(s);
    }
}
