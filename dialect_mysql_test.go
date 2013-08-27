package meddler

import (
	"database/sql"
	"testing"

	// _ "github.com/go-sql-driver/mysql"
)

// In order to run these tests, uncomment the driver above
// and populate mysqlDSN with a valid data source name

var mysqlDSN string = ""

//Example:
//var mysqlDSN string = "tester:tester@tcp(127.0.0.1:3306)/tester"

// CREATE TABLE `accounts` (
//   `accountid` int(64) NOT NULL AUTO_INCREMENT,
//   `email` varchar(255) NOT NULL,
//   `passhash` varbinary(255) NOT NULL,
//   `secretkey` varbinary(255) NOT NULL,

//   `tsadd` TIMESTAMP DEFAULT '0000-00-00 00:00:00',
//   `tsmod` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
//   `lastlogin` TIMESTAMP DEFAULT '0000-00-00 00:00:00',

//   UNIQUE (`email`),
//   PRIMARY KEY (`accountid`)
// ) ENGINE=XtraDB DEFAULT CHARSET=utf8;
var schemaMySQLAccount = `CREATE TABLE accounts (
 accountid int(64) NOT NULL AUTO_INCREMENT,
 email varchar(255) NOT NULL,
 secretkey varbinary(255) NOT NULL,
 
 tsadd TIMESTAMP DEFAULT '0000-00-00 00:00:00',
 tsmod TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
 lastlogin TIMESTAMP DEFAULT '0000-00-00 00:00:00',
 
 UNIQUE (email),
 PRIMARY KEY (accountid)
 ) ENGINE=InnoDB DEFAULT CHARSET=utf8;
`

func GetMySQL(bot BenchOrTest) (*sql.DB, func()) {

	db, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		// t.Errorf("954509733 FATAL ERROR: CANNOT CREATE DB:%v", err)
		s := "9371673535 Cannot load MySQL driver, skipping MySQL tests"
		bot.Skip(s)
		return nil, func() {}
	}

	if err := db.Ping(); err != nil {
		bot.Fatalf("9335842319 db.Ping failure:%v", err)
	}

	if _, err = db.Exec("DROP TABLE IF EXISTS `accounts`;"); err != nil {
		bot.Fatalf("9335842310 db.Exec failure:%v", err)
	}

	if _, err = db.Exec(schemaMySQLAccount); err != nil {
		bot.Fatalf("9335842311 db.Exec failure:%v", err)
	}

	return db, func() { db.Close() }
}

func TestSaveLoadMySQL(t *testing.T) {
	db, cleanup := GetMySQL(t)
	defer cleanup()

	saveLoadAccount(t, db)
}

func BenchmarkAccountLoadMySQLJustQuery(b *testing.B) {
	db, cleanup := GetMySQL(b)
	defer cleanup()
	pk := insertFakeAccount(b, db)
	accountPtr := new(Account)
	benchmarkLoadAccount(b, db, accountPtr, pk)
}
func BenchmarkAccountLoadMySQLStmtCache(b *testing.B) {
	db, cleanup := GetMySQL(b)
	defer cleanup()
	pk := insertFakeAccount(b, db)
	cache := NewPermStmtCache(db)
	accountPtr := new(Account)
	benchmarkLoadAccount(b, cache, accountPtr, pk)
}

func BenchmarkAccountSaveMySQLJustQuery(b *testing.B) {
	db, cleanup := GetMySQL(b)
	defer cleanup()
	benchmarkSaveAccount(b, db)
}
func BenchmarkAccountSaveMySQLStmtCache(b *testing.B) {
	db, cleanup := GetMySQL(b)
	defer cleanup()
	cache := NewPermStmtCache(db)
	benchmarkSaveAccount(b, cache)
}
