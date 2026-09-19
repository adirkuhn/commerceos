// Package db wraps gorm.Open with connection-pool limits every
// commerceos process must set -- see Open's doc comment for why.
package db

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open connects to Postgres and caps the pool at maxOpenConns.
// database/sql's default pool is unbounded per process; every commerceos
// process (cmd/api, relay, webhook-processor, webhook-forwarder,
// scheduler, and the one-off backfill/repair scripts) connects to the
// same shared Postgres instance, so with no cap here, several always-on
// processes under load eventually exhaust the server's max_connections --
// this is not hypothetical, it took production down (SQLSTATE 53300,
// "too many clients already" / "remaining connection slots are reserved
// for roles with the SUPERUSER attribute").
//
// Callers pick maxOpenConns so every process's cap, summed across the
// whole deployment, stays comfortably under the server's max_connections
// (minus superuser_reserved_connections) -- there's no single safe
// default across such different process shapes (cmd/api serves
// concurrent HTTP requests; a backfill script runs once, alone).
func Open(dsn string, maxOpenConns int) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	// MaxIdleConns deliberately lower than maxOpenConns (never more than
	// half, at least 2) rather than equal to it: equal would let every
	// connection a burst opened sit in the pool indefinitely (bounded only
	// by ConnMaxIdleTime), whereas a smaller idle cap closes a connection
	// the moment it's returned to an already-full idle pool, no waiting on
	// the idle timer -- a connection that isn't actively reused is closed,
	// not held "just in case." ConnMaxIdleTime is the backstop for the
	// idle ones that do get kept: closed within a couple minutes of their
	// last use either way.
	maxIdleConns := maxOpenConns / 2
	if maxIdleConns < 2 {
		maxIdleConns = 2
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	return gormDB, nil
}
