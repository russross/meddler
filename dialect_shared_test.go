package meddler

import (
	"bytes"
	"database/sql"
	"testing"
	"time"
)

// Run all benchmarks with the following command:
// go test -test.bench .
//

type Account struct {
	AccountId int64     `meddler:"accountid,pk"`
	Email     string    `meddler:"email"`
	SecretKey []byte    `meddler:"secretkey"`
	Created   time.Time `meddler:"tsadd,utctimez"` // time.Time doesn't work unless parseTime=true is set in your DSN
	Modified  time.Time `meddler:"tsmod,utctimez"`
	LastLogin time.Time `meddler:"lastlogin,utctimez"`
}

func accountsEqual(bot BenchOrTest, elt *Account, ref *Account) {
	if elt == nil {
		bot.Errorf("Account %s is nil", ref.Email)
		return
	}
	if elt.AccountId != ref.AccountId {
		bot.Errorf("Account %s ID is %v, expected %v", ref.Email, elt.AccountId, ref.AccountId)
	}
	if elt.Email != ref.Email {
		bot.Errorf("Account %s Email is %v, expected %v", ref.Email, elt.Email, ref.Email)
	}
	if bytes.Equal(ref.SecretKey, elt.SecretKey) == false {
		bot.Errorf("Account %s SecretKey is %v, expected %v", ref.Email, elt.SecretKey, ref.SecretKey)
	}
	if !elt.Created.Equal(ref.Created) {
		bot.Errorf("Account %s Created is %v, expected %v", ref.Email, elt.Created, ref.Created)
	}
	if !elt.Modified.Equal(ref.Modified) {
		bot.Errorf("Account %s Modified is %v, expected %v", ref.Email, elt.Modified, ref.Modified)
	}
	if !elt.LastLogin.Equal(ref.LastLogin) {
		bot.Errorf("Account %s LastLogin is %v, expected %v", ref.Email, elt.LastLogin, ref.LastLogin)
	}
}

type BenchOrTest interface {
	Skip(args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func benchmarkLoadAccount(b *testing.B, db DB, accountPtr *Account, pk int) {
	for i := 0; i < b.N; i++ {
		if err := Load(db, "accounts", accountPtr, pk); err != nil {
			b.Fatalf("990847081 Load error on some@example.com: %v", err)
			return
		}
	}
	b.StopTimer()
	db.Exec("delete from accounts")
}
func benchmarkSaveAccount(b *testing.B, db DB) {
	someAccount := &Account{
		AccountId: 0,
		Email:     "benchy@bench.com",
		SecretKey: []byte("12345"),
		Created:   when,
		Modified:  when,
		LastLogin: when,
	}

	for i := 0; i < b.N; i++ {
		if i == 0 {
			// first time, we just insert
			if err := Insert(db, "accounts", someAccount); err != nil {
				b.Fatalf("990847085 DB error on Insert benchy@bench.com:", err)
				return
			}
		} else {
			// i > 0, we update
			if err := Update(db, "accounts", someAccount); err != nil {
				b.Fatalf("990847086 DB error on Update benchy@bench.com:", err)
				return
			}
		}
	}
	b.StopTimer()
	db.Exec("delete from accounts")
}

func insertFakeAccount(bot BenchOrTest, db *sql.DB) int {
	someAccount := &Account{
		AccountId: 0,
		Email:     "some@example.com",
		SecretKey: []byte("12345"),
		Created:   when,
		Modified:  when,
		LastLogin: when,
	}

	if err := Save(db, "accounts", someAccount); err != nil {
		bot.Error("9335842312 DB error on Save:", err)
	}
	return int(someAccount.AccountId)
}

func saveLoadAccount(bot BenchOrTest, db *sql.DB) {
	chris := &Account{
		AccountId: 0,
		Email:     "chris@chris.com",
		SecretKey: []byte("abc"),
		Created:   when,
		Modified:  when,
		LastLogin: when,
	}

	if err := Save(db, "accounts", chris); err != nil {
		bot.Error("952809110 DB error on Save:", err)
	}

	if chris.AccountId < 0 {
		bot.Error("952809111 Expected non-zero, positive primary key, got %v", chris.AccountId)
	}

	newbie := new(Account)

	if err := Load(db, "accounts", newbie, int(chris.AccountId)); err != nil {
		bot.Fatal("952809112 DB error on Load:", err)
	}

	accountsEqual(bot, newbie, chris)

	db.Exec("DELETE FROM accounts")
}
