// Copyright 2015 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package idgener

import (
	"database/sql"
	"github.com/someonegg/goutility/dbutil"
	"golang.org/x/net/context"
)

type sqlGener struct {
	db *dbutil.SQLDB
	tn string
}

func NewSqlGener(driver, dsn, tn string,
	maxConcurrent int) (IDGener, error) {

	_db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	err = _db.Ping()
	if err != nil {
		return nil, err
	}
	_, err = _db.Exec(`CREATE TABLE IF NOT EXISTS ? (
		id BIGINT AUTO_INCREMENT NOT NULL,
		PRIMARY KEY (id)
		)`, tn)
	if err != nil {
		return nil, err
	}
	_db.Exec("INSERT INTO ? (id) VALUES (0)", tn)

	db := dbutil.NewSQLDB(_db, maxConcurrent)
	return &sqlGener{db: db, tn: tn}, nil
}

func (g *sqlGener) Close() error {
	return g.db.Close()
}

func (g *sqlGener) GenID(ctx context.Context) (int64, error) {
	res, err := g.db.Exec(ctx, "INSERT INTO ? () VALUES ()", g.tn)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id > 0 {
		g.db.Exec(ctx, "DELETE FROM ? WHERE id = ?", g.tn, id-1)
	}
	return id, nil
}
