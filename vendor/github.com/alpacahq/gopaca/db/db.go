package db

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	db   *gorm.DB
	once sync.Once
)

const (
	ForShare  = "FOR SHARE"
	ForUpdate = "FOR UPDATE"
)

// DB is a singleton wrapper to the gorm database object.
func DB() *gorm.DB {
	var err error

	once.Do(func() {
		db, err = NewDB()
		if err != nil {
			log.Panic("database initialization failure", "error", err)
		}
	})

	return db
}

/*
Optionally pass in a map of options, such as:
	[PGHOST]localhost
	[PGUSER]postgres
	[PGDATABASE]testdb

These will override the settings made via environment variables
*/
func NewDB(OptionsList ...map[string]string) (dbT *gorm.DB, err error) {
	/*
		Set connection parameters
	*/
	sslmode := env.GetVar("PGSSLMODE")
	host := env.GetVar("PGHOST")
	user := env.GetVar("PGUSER")
	dbname := env.GetVar("PGDATABASE")
	password := env.GetVar("PGPASSWORD")
	logDBString := env.GetVar("LOG_DB")
	maxOpenConns := env.GetVar("DB_MAX_OPEN_CONNS")
	maxIdleConns := env.GetVar("DB_MAX_IDLE_CONNS")

	if len(OptionsList) != 0 {
		options := OptionsList[0]
		for key, val := range options {
			switch key {
			case "PGHOST":
				host = val
			case "PGUSER":
				user = val
			case "PGDATABASE":
				dbname = val
			case "PGPASSWORD":
				password = val
			case "SSLMODE":
				sslmode = val
			case "LOG_DB":
				logDBString = val
			case "DB_MAX_OPEN_CONNS":
				maxOpenConns = val
			case "DB_MAX_IDLE_CONNS":
				maxIdleConns = val
			}
		}
	}

	if sslmode == "" {
		sslmode = "disable"
	}

	params := fmt.Sprintf(
		"host=%v user=%v dbname=%v sslmode=%v password=%v",
		host, user, dbname, sslmode, password,
	)

	dbT, err = gorm.Open("postgres", params)
	if err != nil {
		return nil, err
	}

	// default = 20 (Go's default is 0 == unlimited)
	dbT.DB().SetMaxOpenConns(20)
	if maxOpenConns != "" {
		nMaxOpenConns, err := strconv.Atoi(maxOpenConns)
		if err != nil {
			log.Warn("parse error DB_MAX_OPEN_CONNS", "error", err)
		} else {
			dbT.DB().SetMaxOpenConns(nMaxOpenConns)
			log.Info("set max open connections", "value", nMaxOpenConns)
		}
	} else {
		log.Info("no DB_MAX_OPEN_CONNS, defaults to 20")
	}

	if maxIdleConns != "" {
		nMaxIdleConns, err := strconv.Atoi(maxIdleConns)
		if err != nil {
			log.Warn("parse error DB_MAX_IDLE_CONNS", "error", err)
		} else {
			dbT.DB().SetMaxIdleConns(nMaxIdleConns)
			log.Info("set max idle connections", "value", nMaxIdleConns)
		}
	}

	// so it doesn't reuse stale connections
	dbT.DB().SetConnMaxLifetime(30 * time.Minute)

	// enable logging
	logDB, _ := strconv.ParseBool(logDBString)
	dbT.LogMode(logDB)

	return dbT, nil
}

// MockDB mocks the database using sqlmock.
// Used for testing only.
func MockDB() sqlmock.Sqlmock {
	_, mock, err := sqlmock.NewWithDSN("sqlmock_db_0")
	if err != nil {
		panic("Failed to mock db")
	}
	db, err = gorm.Open("sqlmock", "sqlmock_db_0")
	if err != nil {
		panic("Failed to open mocked db")
	}
	return mock
}

// Reconnect pings the database to re-establish
// a connection.
func Reconnect() error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	return db.DB().Ping()
}

// IsConnectionError returns true if the supplied error
// is a connection related error based on PostgreSQL
// connection exceptions class. See:
// http://www.postgresql.org/docs/9.4/static/errcodes-appendix.html#ERRCODES-TABLE
// for details.
func IsConnectionError(err error) bool {
	return pqErrorCode(err) == "08"
}

func InsufficientResources(err error) bool {
	return pqErrorCode(err) == "53"
}

func pqErrorCode(err error) pq.ErrorCode {
	if err != nil {
		pqErr, ok := err.(*pq.Error)

		if ok {
			return pqErr.Code[0:2]
		}
	}
	return ""
}

// IsSerializabilityError returns true if the supplied error
// is due to a serializability failure in the DB.
func IsSerializabilityError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "could not serialize access due to concurrent update")
}

// Serializable begins a transaction with isolation level
// set to SERIALIZABLE.
func Serializable() *gorm.DB {
	return DB().Begin().Exec("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
}

// RepeatableRead begins a transaction with isolation level
// set to REPEATABLE READ.
func RepeatableRead() *gorm.DB {
	return DB().Begin().Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
}

// ReadCommitted begins a transaction with isolation level
// set to READ COMMITTED.
func ReadCommitted() *gorm.DB {
	return DB().Begin().Exec("SET TRANSACTION ISOLATION LEVEL READ COMMITTED")
}

// ReadUncomitted begins a transaction with isolation level
// set to READ UNCOMMITTED.
func ReadUncomitted() *gorm.DB {
	return DB().Begin().Exec("SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED")
}

// Begin a transaction.
func Begin() *gorm.DB {
	return DB().Begin()
}
