package meddler

import (
	"database/sql"
	"testing"

	// _ "github.com/lib/pq"
)

// In order to run these tests, uncomment the driver above
// and populate postgresDSN with a valid data source name

var postgresConnParams string = ""

// Example:
//var postgresConnParams string = "user=tester password=tester dbname=tester host=127.0.0.1 port=5432"

var schemaPostgresAccount = `
DROP TABLE IF EXISTS accounts;
CREATE TABLE accounts (
    accountid bigserial NOT NULL PRIMARY KEY,
    email varchar(255) NOT NULL,
    passhash bytea NOT NULL DEFAULT E'\\xDEFA1745',
    secretkey bytea NOT NULL DEFAULT E'\\xDEFA1745',
    
    tsadd TIMESTAMP NOT NULL,
    tsmod TIMESTAMP NOT NULL,
    lastlogin TIMESTAMP NOT NULL DEFAULT 'epoch'
);
`

func GetPostgres(bot BenchOrTest) (*sql.DB, func()) {

	db, err := sql.Open("postgres", postgresConnParams)
	if err != nil {
		// t.Errorf("954509733 FATAL ERROR: CANNOT CREATE DB:%v", err)
		s := "9034641821 Cannot load Postgres driver, skipping Postgres tests"
		bot.Skip(s)
		return nil, func() {}
	}

	if err := db.Ping(); err != nil {
		bot.Fatalf("9034641822 db.Ping failure:%v", err)
	}

	if _, err = db.Exec(schemaPostgresAccount); err != nil {
		bot.Fatalf("9034641824 db.Exec failure:%v", err)
	}

	Quote = `"`
	Placeholder = "$1"
	PostgreSQL = true

	return db, func() {
		db.Close()
		Quote = "`"
		Placeholder = "?"
		PostgreSQL = false
	}
}

func TestSaveLoadPostgres(t *testing.T) {
	db, cleanup := GetPostgres(t)
	defer cleanup()

	saveLoadAccount(t, db)
}

func BenchmarkAccountLoadPostgresJustQuery(b *testing.B) {
	db, cleanup := GetPostgres(b)
	defer cleanup()
	pk := insertFakeAccount(b, db)
	accountPtr := new(Account)
	benchmarkLoadAccount(b, db, accountPtr, pk)
}
func BenchmarkAccountLoadPostgresStmtCache(b *testing.B) {
	db, cleanup := GetPostgres(b)
	defer cleanup()
	pk := insertFakeAccount(b, db)
	cache := NewPermStmtCache(db)
	accountPtr := new(Account)
	benchmarkLoadAccount(b, cache, accountPtr, pk)
}

func BenchmarkAccountSavePostgresJustQuery(b *testing.B) {
	db, cleanup := GetPostgres(b)
	defer cleanup()
	benchmarkSaveAccount(b, db)
}
func BenchmarkAccountSavePostgresStmtCache(b *testing.B) {
	db, cleanup := GetPostgres(b)
	defer cleanup()
	cache := NewPermStmtCache(db)
	benchmarkSaveAccount(b, cache)
}
