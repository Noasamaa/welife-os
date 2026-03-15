// Package sqlitevec registers the sqlite-vec extension with every SQLite
// connection opened by go-sqlcipher. It embeds the sqlite-vec C amalgamation
// and calls sqlite3_auto_extension during init() so the vec0 virtual table
// is available on all new connections—including encrypted ones.
//
// The .c and .h files in this directory are vendored from:
//   - sqlite3.h / sqlite3ext.h: SQLite 3.33.0 (matching go-sqlcipher v4.4.2)
//   - sqlite-vec.h / sqlite-vec.c: sqlite-vec v0.1.6
package sqlitevec

/*
#cgo CFLAGS: -DSQLITE_VEC_STATIC -std=c99
#cgo LDFLAGS: -lm

// Forward declarations instead of #include "sqlite-vec.h" to avoid
// sqlite3ext.h macro redefinitions of sqlite3_auto_extension.
typedef struct sqlite3 sqlite3;
typedef struct sqlite3_api_routines sqlite3_api_routines;

// Defined in sqlite-vec.c (compiled by CGO from this package).
extern int sqlite3_vec_init(sqlite3 *db, char **pzErrMsg,
                            const sqlite3_api_routines *pApi);

// Defined in go-sqlcipher's embedded SQLite (resolved at link time).
extern int sqlite3_auto_extension(void (*xEntryPoint)(void));

static int register_vec(void) {
    return sqlite3_auto_extension((void (*)(void))sqlite3_vec_init);
}
*/
import "C"

import "fmt"

func init() {
	rc := C.register_vec()
	if rc != 0 {
		panic(fmt.Sprintf("sqlitevec: sqlite3_auto_extension failed with rc=%d", rc))
	}
}
