// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbutil

import (
	. "database/sql"
	"github.com/someonegg/goutility/chanutil"
	"golang.org/x/net/context"
)

// SQLStmt is a contexted sql Stmt.
type SQLStmt struct {
	db *SQLDB
	s  *Stmt
}

func newSQLStmt(db *SQLDB, s *Stmt) *SQLStmt {
	return &SQLStmt{db: db, s: s}
}

func (s *SQLStmt) Close() error {
	return s.s.Close()
}

func (s *SQLStmt) Exec(ctx context.Context,
	args ...interface{}) (Result, error) {

	err := s.db.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer s.db.releaseConn()

	return s.s.Exec(args...)
}

func (s *SQLStmt) Query(ctx context.Context,
	args ...interface{}) (*Rows, error) {

	err := s.db.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer s.db.releaseConn()

	return s.s.Query(args...)
}

func (s *SQLStmt) QueryRow(ctx context.Context,
	args ...interface{}) (*Row, error) {

	err := s.db.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer s.db.releaseConn()

	return s.s.QueryRow(args...), nil
}

// SQLTx is a contexted sql Tx.
type SQLTx struct {
	db *SQLDB
	*Tx
}

func newSQLTx(db *SQLDB, tx *Tx) *SQLTx {
	return &SQLTx{db: db, Tx: tx}
}

func (tx *SQLTx) Commit() error {
	defer tx.db.releaseConn()

	return tx.Tx.Commit()
}

func (tx *SQLTx) Rollback() error {
	defer tx.db.releaseConn()

	return tx.Tx.Rollback()
}

// SQLDB is a contexted sql DB.
type SQLDB struct {
	db     *DB
	concur chanutil.Semaphore
}

func NewSQLDB(db *DB, maxConcurrent int) *SQLDB {
	mi := maxConcurrent / 5
	if mi <= 0 {
		mi = 2
	}

	db.SetMaxIdleConns(mi)

	d := &SQLDB{}
	d.db = db
	if maxConcurrent > 0 {
		d.concur = chanutil.NewSemaphore(maxConcurrent)
	}
	return d
}

func (d *SQLDB) acquireConn(ctx context.Context) error {
	if d.concur == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	// Acquire
	case d.concur <- struct{}{}:
		return nil
	}
}

func (d *SQLDB) releaseConn() {
	if d.concur == nil {
		return
	}

	<-d.concur
}

func (d *SQLDB) Close() error {
	return d.db.Close()
}

func (d *SQLDB) Begin(ctx context.Context) (*SQLTx, error) {
	success := false

	err := d.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !success {
			d.releaseConn()
		}
	}()

	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	success = true
	return newSQLTx(d, tx), nil
}

func (d *SQLDB) Prepare(ctx context.Context,
	query string) (*SQLStmt, error) {

	err := d.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer d.releaseConn()

	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, err
	}

	return newSQLStmt(d, stmt), nil
}

func (d *SQLDB) Exec(ctx context.Context,
	query string, args ...interface{}) (Result, error) {

	err := d.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer d.releaseConn()

	return d.db.Exec(query, args...)
}

func (d *SQLDB) Ping(ctx context.Context) error {
	err := d.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer d.releaseConn()

	return d.db.Ping()
}

func (d *SQLDB) Query(ctx context.Context,
	query string, args ...interface{}) (*Rows, error) {

	err := d.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer d.releaseConn()

	return d.db.Query(query, args...)
}

func (d *SQLDB) QueryRow(ctx context.Context,
	query string, args ...interface{}) (*Row, error) {

	err := d.acquireConn(ctx)
	if err != nil {
		return nil, err
	}
	defer d.releaseConn()

	return d.db.QueryRow(query, args...), nil
}
