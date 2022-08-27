package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/caleberi/gostripe/internal/driver"
	"github.com/caleberi/gostripe/internal/models"
)

const version = "1.0.0"

// configuration setup for the application which allows application
// information management  retrieved from the system environment or configuration file
type config struct {
	port int
	env  string
	db   struct {
		dns string
	}
	stripe struct {
		key    string
		secret string
	}
}

// creates basic setup properties for tha application which might be needed along the way
type application struct {
	config   config
	infoLog  *log.Logger
	errorLog *log.Logger
	version  string
	DB       models.DBModel
}

// serve function basically start the application server via `net/http`
// Server construct
func (app *application) serve() error {

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	app.infoLog.Printf("starting API server on port %[1]d with url: http://localhost:%[1]d  ...", app.config.port)
	return srv.ListenAndServe()
}

func main() {

	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "📌 app server port")
	flag.StringVar(&cfg.env, "environment", "development", "📌 application runtime environment {production|developement|maintenace}")
	flag.StringVar(&cfg.db.dns, "dsn", "caleb:secret@tcp(localhost:3306)/gostripe?parseTime=true&tls=false", "📌 database domain service name (DSN)")

	flag.Parse()

	// retrieve stripe setup from os package
	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	infoLog.Printf("db.dns :- %s ", cfg.db.dns)
	// connect to the database
	conn, err := driver.OpenDB(cfg.db.dns)
	if err != nil {
		errorLog.Fatalln(err)
	}
	defer conn.Close()

	// initializing the application with obtained configuration
	app := &application{
		config:   cfg,
		infoLog:  infoLog,
		errorLog: errorLog,
		version:  version,
		DB: models.DBModel{
			DB: conn,
		},
	}

	if err := app.serve(); err != nil {
		app.errorLog.Fatalln(err)
	}

}
