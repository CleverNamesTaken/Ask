package ask_db

import (
	"database/sql"
	"fmt"
	"go-ask/internal/debug"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

var DB *sql.DB
var Driver string

func TableExists(db *sql.DB, tableName string) bool {
	// Check if a table exists
	var exists bool
	var query string
	debug.Print("[*] Checking table exists for driver: %s %s", tableName, Driver)
	if Driver == "mysql" {
		query = `
			SELECT COUNT(*)
			FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name = ?
		`
	} else {
		query = `
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?;
		`
	}

	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		fmt.Printf("[!] Table %s does not exist.\n", tableName)
		return false
	}
	return exists
}

func Init() (db *sql.DB, err error) {
	//initialize connection to the database
	debug.Print("[*] Reading config file to connect to the database")
	if DB != nil {
		return db, nil // already connected
	}

	port := viper.GetString("port")
	// Create the database if it doesn't exist
	if port != "" {
		// If mysql is listed for the connection type, use mysql
		Driver = "mysql"
		debug.Print("[!] Assuming the sql driver is mysql")
		host := viper.GetString("host")
		username := viper.GetString("username")
		password := viper.GetString("password")
		dbname := viper.GetString("database")

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", username, password, host, port)
		debug.Print("[*] Attempting connection to mysql server: %s", dsn)
		DB, err = sql.Open("mysql", dsn)

		debug.Print("[*] Attempting create database if it does not exist: %s%s", dsn, dbname)
		query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbname)
		_, err = DB.Exec(query)
		if err != nil {
			fmt.Fprintf(os.Stdout, "[!] Failed to access database.  Check your configuration file and the database to ensure access.\n\n%v\n", err)
			os.Exit(1)
		}

		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, dbname)
		debug.Print("[*] Attempting connect to the database: %s", dsn)

		DB, err = sql.Open(Driver, dsn)
		if err != nil {
			return nil, fmt.Errorf("error opening DB: %w", err)
		}
	} else {
		//If mysql is not listed, let's assume it is a sql database
		Driver = "sqlite3"
		debug.Print("[!] Assuming the sql driver is sqlite3")
		sqlFile := viper.GetString("sqlFile")

		if sqlFile[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			sqlFile = strings.Replace(sqlFile, "~", homeDir, 1)
		}

		debug.Print("[*] Attempting connection to sqlite: %s", sqlFile)
		DB, err = sql.Open(Driver, sqlFile)
		if err != nil {
			fmt.Fprintf(os.Stdout, "[!] Failed to access database.  Check your configuration file and the database to ensure access.\n\n%v\n", err)
			os.Exit(1)
		}

	}

	debug.Print("[+] Succesfully opened database")

	if err = DB.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging DB: %w", err)
	}
	debug.Print("[+] Connected to DB")
	return db, nil
}
