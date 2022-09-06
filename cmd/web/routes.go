package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Use(SessionLoader)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Get("/", app.RenderHomePage)
	mux.Get("/virtual-terminal", app.VirtualTerminal)
	mux.Post("/virtual-terminal-payment-succeeded", app.VirtualTerminalPaymentSucceeded)
	mux.Get("/virtual-terminal-receipt", app.VirtualTerminalReceipt)
	mux.Post("/payment-succeeded", app.PaymentSucceeded)
	mux.Get("/plans/bronze-plan", app.RenderBronzePlan)
	mux.Get("/receipt", app.Receipt)
	mux.Get("/receipt/bronze", app.BronzePlanReceipt)
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))
	mux.Get("/widgets/{id}", app.ChargeOnce)

	// auth routes

	mux.Get("/login", app.LoginPage)

	return mux
}
