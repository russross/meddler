package meddler

import (
	"database/sql"
	"fmt"
)

// The high level statement cache interface
type StmtCache interface {
	// Prepare a query string & add to the cache
	// Cache invalidation depends on implementation of interface
	// This will automatically prepare the statement,returning (nil, error)
	// if the associated DB cannot properly prepare the statment
	// If a specific query alread exists, it will return the existing prepared
	// statment rather than re-created the prepared statment.
	//
	// One note on usage: Preparing all known statements ahead of time can
	// help catch SQL errors during startup rather than during runtime.
	PrepareStmtForQuery(query string) (*sql.Stmt, error)

	// Passthrough Exec implementation
	Exec(query string, args ...interface{}) (sql.Result, error)
	// Passthrough Query implementation
	Query(query string, args ...interface{}) (*sql.Rows, error)
	// Passthrough QueryRow implementation
	QueryRow(query string, args ...interface{}) *sql.Row
}

// The simple, naive implementation.
// No cache invalidation whatsoever.
// Useful when the total number of queries is bounded.
//
// Example Usage:
//	cache := NewPermStmtCache()
//	if err = Save(cache, ...); err != nil {}
//	if err = Update(cache, ...); err != nil {}
type PermStmtCache struct {
	db    *sql.DB
	cache map[string]*sql.Stmt
}

func NewPermStmtCache(db *sql.DB) *PermStmtCache {
	sc := new(PermStmtCache)
	sc.db = db
	sc.cache = make(map[string]*sql.Stmt)
	return sc
}

func (sc *PermStmtCache) PrepareStmtForQuery(query string) (*sql.Stmt, error) {

	stmt := sc.cache[query]
	if stmt != nil {
		return stmt, nil
	}

	if sc.db == nil {
		return nil, fmt.Errorf("143766254 PrepareQueryStr sc.db must not be nil")
	}

	stmt, err := sc.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("143766255 PrepareQueryStr sc.db.Prepare failure for query:%v\nerr:%v", query, err)
	}

	sc.cache[query] = stmt
	return stmt, nil
}

func (sc *PermStmtCache) Exec(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := sc.PrepareStmtForQuery(query)
	if err != nil {
		return nil, fmt.Errorf("1437656255 PrepareStmtForQuery failure for query:%v\nerr:%v", query, err)
	}
	result, err := stmt.Exec(args...)
	if err != nil {
		return nil, fmt.Errorf("1437656256 stmt.Exec failure for query:%v\nargs:%v\nerr:%v", query, args, err)
	}
	return result, nil
}
func (sc *PermStmtCache) Query(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := sc.PrepareStmtForQuery(query)
	if err != nil {
		return nil, fmt.Errorf("1750405516 PrepareStmtForQuery failure for query:%v\nerr:%v", query, err)
	}
	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("1750405517 stmt.Exec failure for query:%v\nargs:%v\nerr:%v", query, args, err)
	}
	return rows, nil
}
func (sc *PermStmtCache) QueryRow(query string, args ...interface{}) *sql.Row {
	stmt, err := sc.PrepareStmtForQuery(query)
	if err != nil {
		// we cannot safely create nice errors, so we basically force the db to create an Row populated with an error for us.
		// err = fmt.Errorf("1895395626 PrepareStmtForQuery failure for query:%v\nerr:%v", query, err)
		return sc.db.QueryRow(query, args...)
	}
	return stmt.QueryRow(args...)
}

// A variant of the PermStmtCache for use within one transaction.
// Prepared statements must be "converted" to be used within a transcation with the tx.Stmt() method
//
// Example Usage:
//	cache := NewPermStmtCache()
//	tx, err := db.Begin()
//	txcache := NewTxPermStmtCache(cache, tx)
//	if err = Save(txcache, ...); err != nil {}
//	if err = Update(txcache, ...); err != nil {}
//	if err = tx.Commit(); err != nil {
//		t.Errorf("Commit error: %v", err)
//	}
type TxStmtCache struct {
	tx *sql.Tx
	sc StmtCache
}

func NewTxStmtCache(sc StmtCache, tx *sql.Tx) *TxStmtCache {
	tsc := new(TxStmtCache)
	tsc.tx = tx
	tsc.sc = sc
	return tsc
}

func (tsc *TxStmtCache) Exec(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := tsc.sc.PrepareStmtForQuery(query)
	var txstmt *sql.Stmt
	if err != nil {
		// if the statement was not prepared before the transaction was created,
		// we have to prepare it using our current tx
		txstmt, err = tsc.tx.Prepare(query)
		if err != nil {
			return nil, fmt.Errorf("1984788494 PrepareStmtForQuery failure\nquery:%v\nerr:%v", query, err)
		}
	} else {
		// if the statement was prepared before the transaction was created, no problem,
		// we just need to convert it to a tx-safe statment:
		txstmt = tsc.tx.Stmt(stmt)
	}
	result, err := txstmt.Exec(args...)
	if err != nil {
		return nil, fmt.Errorf("1984788495 stmt.Exec failure\nquery:%v\nargs:%v\nerr:%v", query, args, err)
	}
	return result, nil
}
func (tsc *TxStmtCache) Query(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := tsc.sc.PrepareStmtForQuery(query)
	if err != nil {
		return nil, fmt.Errorf("1686396506 PrepareStmtForQuery failure for query:%v\nerr:%v", query, err)
	}
	txstmt := tsc.tx.Stmt(stmt)
	rows, err := txstmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("1686396507 stmt.Exec failure for query:%v\nargs:%v\nerr:%v", query, args, err)
	}
	return rows, nil
}
func (tsc *TxStmtCache) QueryRow(query string, args ...interface{}) *sql.Row {
	stmt, err := tsc.sc.PrepareStmtForQuery(query)
	if err != nil {
		// we cannot safely create *sql.Row with an embedded error.
		// further more, QueryRow is guarunteed to return a non-nil value,
		// so we basically force the db to create an Row populated with an error for us.
		return tsc.tx.QueryRow(query, args...)
	}
	txstmt := tsc.tx.Stmt(stmt)
	return txstmt.QueryRow(args...)
}
