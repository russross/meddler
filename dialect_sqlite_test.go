package meddler

import (
	"database/sql"
	"testing"

	// _ "github.com/mattn/go-sqlite3"
)

// In order to run these tests, uncomment the driver above

var schemaSqliteAccount = `CREATE TABLE accounts (
    accountid INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    secretkey BLOB NOT NULL,
 
    tsadd DATETIME,
    tsmod DATETIME,
    lastlogin DATETIME
 )`

func GetSqlite(bot BenchOrTest) (*sql.DB, func()) {

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		s := "9624948125 Cannot load Sqlite driver, skipping Sqlite tests"
		bot.Skip(s)
		return nil, func() {}
	}

	if err := db.Ping(); err != nil {
		bot.Fatalf("9624948126 db.Ping failure:%v", err)
	}

	if _, err = db.Exec("DROP TABLE IF EXISTS `accounts`;"); err != nil {
		bot.Fatalf("9624948127 db.Exec failure:%v", err)
	}

	if _, err = db.Exec(schemaSqliteAccount); err != nil {
		bot.Fatalf("9624948128 db.Exec failure:%v", err)
	}

	return db, func() { db.Close() }
}

func TestSaveLoadSqlite(t *testing.T) {
	db, cleanup := GetSqlite(t)
	defer cleanup()

	saveLoadAccount(t, db)
}

func BenchmarkAccountLoadSqliteJustQuery(b *testing.B) {
	db, cleanup := GetSqlite(b)
	defer cleanup()
	pk := insertFakeAccount(b, db)
	accountPtr := new(Account)
	benchmarkLoadAccount(b, db, accountPtr, pk)
}
func BenchmarkAccountLoadSqliteStmtCache(b *testing.B) {
	db, cleanup := GetSqlite(b)
	defer cleanup()
	pk := insertFakeAccount(b, db)
	cache := NewPermStmtCache(db)
	accountPtr := new(Account)
	benchmarkLoadAccount(b, cache, accountPtr, pk)
}

func BenchmarkAccountSaveSqliteJustQuery(b *testing.B) {
	db, cleanup := GetSqlite(b)
	defer cleanup()
	benchmarkSaveAccount(b, db)
}
func BenchmarkAccountSaveSqliteStmtCache(b *testing.B) {
	db, cleanup := GetSqlite(b)
	defer cleanup()
	cache := NewPermStmtCache(db)
	benchmarkSaveAccount(b, cache)
}
