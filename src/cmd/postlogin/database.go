//
// The MIT License (MIT)
//
// Copyright (c) 2017 Michael Tratz <support@esosoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package main

import (
	"log"
	"strconv"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
	"time"
)

type Database struct {
	conn     *sql.DB
	queries map[string]*sql.Stmt
}

func connectDatabase() (*Database, error) {

	// Connect to Database
	host := "@tcp(" + Config.DB.Host + ":" + strconv.FormatUint(uint64(Config.DB.Port), 10) + ")/"
	if Config.DB.Socket != "" {
		host = "@unix(" + Config.DB.Socket + ")/"
	}

	db_conn, err := sql.Open("mysql", Config.DB.User + ":" + Config.DB.Password + host + Config.DB.Name + "?" + Config.DB.Options)
        if err != nil {
		return nil, err
	}

        err = db_conn.Ping()
        if err != nil {
		return nil, err
        }

	// Set database connection settings
	timeout, err := time.ParseDuration(Config.DB.ConnectionMaxLifeTime)
	if err != nil {
		timeout = 0
	}
	db_conn.SetConnMaxLifetime(timeout)
	db_conn.SetMaxOpenConns(Config.DB.MaxOpenConnections)
	db_conn.SetMaxIdleConns(Config.DB.MaxIdleConnections)


	var db Database = Database{conn: db_conn, queries: make(map[string]*sql.Stmt)}

	// Prepare SQL Queries
	for name, sql := range Config.DB.Queries {
		db.queries[name], err = db.conn.Prepare(sql.Sql)
		if err != nil {
			return nil, fmt.Errorf("Unable to prepare query '%s': %v", name, err)
		}
	}

	return &db, nil
}

func (db *Database) addIP(user *User) error {

	var (
		mbxid uint32
		auth_id int64
	)
	query := db.queries

	// Get mailbox id
	err := query["select_mbx_id"].QueryRow(user.username, user.domain).Scan(&mbxid)
	if err != nil {
		return err
	}
	if *debug {
		log.Println("[DEBUG] Query Mailbox ID:", mbxid)
	}
	
	// Get or insert into auth relay ip table
	err = query["select_auth_relay_ip_id"].QueryRow(mbxid).Scan(&auth_id)
	switch {
	case err == sql.ErrNoRows:
		res, err := query["insert_auth_relay_ip"].Exec(mbxid, user.ip)
		if err != nil {
			return err
		}
		if *debug {
			auth_id, err = res.LastInsertId()
			log.Println("[DEBUG] Inserted Auth Relay IP: ", mbxid, auth_id)
		}
		return nil
	case err != nil:
		return err
	}

	if *debug {
		log.Println("[DEBUG] Update Auth Relay IP:", auth_id)
	}

	// Update auth relay ip table
	_, err = query["update_auth_relay_ip"].Exec(user.ip, auth_id)
	if err != nil {
		return err
	}
	
	return nil
}

