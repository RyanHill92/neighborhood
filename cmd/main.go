package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// Config holds app secrets.
type Config struct {
	dbUser     string
	dbPassword string
	dbHost     string
	dbName     string
}

const (
	dsnFmt        string = "%s:%s@tcp(%s)/%s?parseTime=true"
	dbUserKey     string = "DB_USER"
	dbPasswordKey string = "DB_PASSWORD"
	dbHostKey     string = "DB_HOST"
	dbNameKey     string = "DB_NAME"
)

var requiredEnv = []string{
	dbUserKey,
	dbPasswordKey,
	dbHostKey,
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := run(); err != nil {
		log.Println("error running app", err)
		os.Exit(1)
	}
}

func run() error {
	for _, key := range requiredEnv {
		if _, ok := os.LookupEnv(key); !ok {
			return fmt.Errorf("must set %s", key)
		}
	}

	config := getConfig()

	dsn := fmt.Sprintf(dsnFmt, config.dbUser, config.dbPassword, config.dbHost, config.dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("error opening DB connection: %w", err)
	}

	defer func() {
		log.Println("main: closing database")
		db.Close()
	}()

	for {
		time.Sleep(200 * time.Millisecond)
		if db.Ping() == nil {
			break
		}
	}

	store, err := NewMySQLStore(db)
	if err != nil {
		return fmt.Errorf("error initializing store: %w", err)
	}

	defer func() {
		log.Println("main: closing store")
		store.Close()
	}()

	handler := &handler{store: store}

	router := mux.NewRouter()
	router.HandleFunc("/", handler.ReportHealth)
	router.HandleFunc("/houses", handler.GetAllHouses).Methods("GET")
	router.HandleFunc("/trees/{houseID}", handler.GetTreesByHouseID).Methods("GET")
	router.HandleFunc("/trees/{houseID}", handler.AddTreeByHouseID).Methods("POST")
	router.HandleFunc("/trees/{treeID}", handler.RemoveTreeByTreeID).Methods("DELETE")
	router.HandleFunc("/storm/{houseID}", handler.SendStormByHouseID).Methods("POST")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	serverErrors := make(chan error, 1)

	go func() {
		log.Println("web service listening on port 5000")
		serverErrors <- http.ListenAndServe(":5000", router)
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-shutdown:
		log.Println("main: received shutdown signal")
	}

	return nil
}

func getConfig() Config {
	config := Config{}
	config.dbUser = os.Getenv(dbUserKey)
	config.dbPassword = os.Getenv(dbPasswordKey)
	config.dbHost = os.Getenv(dbHostKey)
	config.dbName = os.Getenv(dbNameKey)
	return config
}
