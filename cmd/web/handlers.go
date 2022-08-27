package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/caleberi/gostripe/internal/cards"
	"github.com/caleberi/gostripe/internal/models"
	"github.com/go-chi/chi/v5"
)

type TransactionData struct {
	FirstName       string
	LastName        string
	Email           string
	PaymentIntentID string
	PaymentMethodID string
	PaymentAmount   int
	PaymentCurrency string
	LastFour        string
	ExpiryMonth     int
	ExpiryYear      int
	BankReturnCode  string
}

func (app *application) GetTransactionData(r *http.Request) (TransactionData, error) {
	var tx TransactionData
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return tx, err
	}

	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
	email := r.Form.Get("cardholder_email")
	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	amount, _ := strconv.Atoi(paymentAmount)

	// add validation to the incoming data

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.RetrivePaymentIntent(paymentIntent)

	if err != nil {
		app.errorLog.Println(err)
		return tx, nil
	}

	pm, err := card.GetPaymentMethod(paymentMethod)

	if err != nil {
		app.errorLog.Println(err)
		return tx, err
	}

	lastFour := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear

	tx = TransactionData{
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		PaymentIntentID: paymentIntent,
		PaymentAmount:   amount,
		LastFour:        lastFour,
		ExpiryMonth:     int(expiryMonth),
		ExpiryYear:      int(expiryYear),
		PaymentCurrency: paymentCurrency,
		BankReturnCode:  pi.Charges.Data[0].ID,
	}
	return tx, nil
}

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "terminal", &templateData{}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) VirtualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {

	tx, err := app.GetTransactionData(r)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	txn := models.Transaction{
		Amount:              tx.PaymentAmount,
		Currency:            tx.PaymentCurrency,
		LastFour:            tx.LastFour,
		ExpiryMonth:         tx.ExpiryMonth,
		ExpiryYear:          tx.ExpiryYear,
		BankReturnCode:      tx.BankReturnCode,
		TransactionStatusID: 2,
		PaymentMethod:       tx.PaymentMethodID,
		PaymenyIntent:       tx.PaymentIntentID,
	}

	_, err = app.SaveTransaction(txn)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infoLog.Printf("data: [{%v}]", tx)

	app.Session.Put(r.Context(), "receipt", tx)
	http.Redirect(w, r, "/virtual-terminal-receipt", http.StatusSeeOther)

}

func (app *application) RenderHomePage(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "home", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	widgetId, _ := strconv.Atoi(r.Form.Get("product_id"))

	tx, err := app.GetTransactionData(r)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	customerID, err := app.SaveCustomer(tx.FirstName, tx.LastName, tx.Email)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infoLog.Printf(":: Customer with ID : %d created ... ", customerID)

	txn := models.Transaction{
		Amount:              tx.PaymentAmount,
		Currency:            tx.PaymentCurrency,
		LastFour:            tx.LastFour,
		ExpiryMonth:         tx.ExpiryMonth,
		ExpiryYear:          tx.ExpiryYear,
		BankReturnCode:      tx.BankReturnCode,
		TransactionStatusID: 2,
		PaymentMethod:       tx.PaymentMethodID,
		PaymenyIntent:       tx.PaymentIntentID,
	}

	txnID, err := app.SaveTransaction(txn)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infoLog.Printf(":: Transaction with ID : %d created ... ", txnID)

	order := models.Order{
		WidgetID:      widgetId,
		TransactionID: txnID,
		CustomerID:    customerID,
		StatusID:      1,
		Quantity:      1,
		Amount:        tx.PaymentAmount,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = app.SaveOrder(order)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infoLog.Printf("data: [{%v}]", tx)

	app.Session.Put(r.Context(), "receipt", tx)
	http.Redirect(w, r, "/receipt", http.StatusSeeOther)

}

func (app *application) Receipt(w http.ResponseWriter, r *http.Request) {
	tx := app.Session.Get(r.Context(), "receipt").(TransactionData)
	data := make(map[string]interface{})
	data["tx"] = tx
	app.Session.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "receipt", &templateData{Data: data}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) VirtualTerminalReceipt(w http.ResponseWriter, r *http.Request) {
	tx := app.Session.Get(r.Context(), "receipt").(TransactionData)
	data := make(map[string]interface{})
	data["tx"] = tx
	app.Session.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "virtual-terminal-receipt", &templateData{Data: data}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ChargeOnce(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	widgetID, err := strconv.Atoi(id)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	widget, err := app.DB.GetWidget(widgetID)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})
	data["widget"] = widget
	if err := app.renderTemplate(w, r, "buy-one", &templateData{
		Data: data,
	}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) SaveCustomer(firstName, lastName, email string) (int, error) {
	customer := models.Customer{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	id, err := app.DB.InsertCustomer(customer)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (app *application) SaveTransaction(txn models.Transaction) (int, error) {
	id, err := app.DB.InsertTransaction(txn)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (app *application) SaveOrder(order models.Order) (int, error) {
	id, err := app.DB.InsertOrder(order)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (app *application) RenderBronzePlan(w http.ResponseWriter, r *http.Request) {
	widget, err := app.DB.GetWidget(2)
	if err != nil {
		app.errorLog.Printf("%v", err)
		return
	}
	data := make(map[string]interface{})

	data["widget"] = widget
	if err := app.renderTemplate(w, r, "bronze-plan", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Print(err)
	}
}

func (app *application) BronzePlanReceipt(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "receipt-plan", &templateData{}); err != nil {
		app.errorLog.Print(err)
	}
}
