package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/caleberi/gostripe/internal/driver"
	"github.com/caleberi/gostripe/internal/models"
)

const version = "1.0.0"
const cssVersion = "1"

var session *scs.SessionManager

type config struct {
	port int
	env  string
	api  string
	db   struct {
		dns string
	}
	stripe struct {
		key    string
		secret string
	}
}

type application struct {
	config        config
	infoLog       *log.Logger
	errorLog      *log.Logger
	templateCache map[string]*template.Template
	version       string
	DB            models.DBModel
	Session       *scs.SessionManager
}

func (app *application) serve() error {

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	app.infoLog.Printf("starting http server on port %[1]d with url: http://localhost:%[1]d  ...", app.config.port)
	return srv.ListenAndServe()
}

func main() {
	gob.Register(TransactionData{})

	var cfg config

	flag.IntVar(&cfg.port, "port", 3000, "📌 app server port")
	flag.StringVar(&cfg.env, "environment", "development", "📌 application runtime environment")
	flag.StringVar(&cfg.api, "api", "http://localhost:4001", "📌 api endpoint entry for application")
	flag.StringVar(&cfg.db.dns, "dsn", "root:root@tcp(localhost:3306)/gostripe?parseTime=true&tls=false", "📌 database domain service name (DSN)")

	flag.Parse()

	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	conn, err := driver.OpenDB(cfg.db.dns)

	if err != nil {
		errorLog.Fatal(err)
	}

	defer conn.Close()

	session = scs.New()
	session.Lifetime = 24 * time.Hour

	tc := make(map[string]*template.Template)

	app := &application{
		config:        cfg,
		templateCache: tc,
		infoLog:       infoLog,
		errorLog:      errorLog,
		version:       version,
		DB: models.DBModel{
			DB: conn,
		},
		Session: session,
	}

	if err := app.serve(); err != nil {
		app.errorLog.Fatalln(err)
	}
}
