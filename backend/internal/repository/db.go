package repository

import (
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/yourorg/csms-backend/config"
)

var DB *sqlx.DB

func Connect() {
	var err error
	DB, err = sqlx.Connect("mysql", config.Cfg.Database.DSN)
	if err != nil {
		log.Fatalf("[db] connect: %v", err)
	}
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	log.Println("[db] connected")
}
