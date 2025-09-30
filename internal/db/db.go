package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"
)

type MysqlDB struct {
}

func (mysqlDb *MysqlDB) Init() *sql.DB {
	dbConfig := mysql.Config{
		User:      "root",
		Passwd:    "",
		Net:       "tcp",
		Addr:      "127.0.0.1:3306",
		DBName:    "go_task",
		ParseTime: true,
	}

	db, err := sql.Open("mysql", dbConfig.FormatDSN())
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err.Error())
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	log.Printf("Connected to mysql")

	return db
}
