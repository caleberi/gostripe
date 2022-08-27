package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/caleberi/gostripe/internal/cards"
	"github.com/caleberi/gostripe/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v72"
)

// payload representation for stripe
type stripePayload struct {
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
	LastFour      string `json:"last_four"`
	Plan          string `json:"plan"`
	ExpiryMonth   int    `json:"exp_month"`
	ExpiryYear    int    `json:"exp_year"`
	CardBrand     string `json:"card_brand"`
	ProductID     string `json:"product_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

// how we expect our response to be after every response has been generated
type jsonResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      string `json:"id,omitempty"`
}

// process each payment intent request
func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {

	var payload stripePayload
	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infoLog.Printf("payload :.. -> %v", payload)

	// amount conversion for money
	amount, err := strconv.Atoi(payload.Amount)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// build card with secrets
	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}

	ok := true

	paymentIntent, msg, err := card.Charge(payload.Currency, amount)

	if err != nil {
		ok = false
	}

	if ok {
		out, err := json.MarshalIndent(paymentIntent, "", "  ")
		if err != nil {
			app.errorLog.Println(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	} else {
		j := jsonResponse{
			OK:      false,
			Message: msg,
			Content: "",
		}
		out, err := json.MarshalIndent(j, "", "  ")
		if err != nil {
			app.errorLog.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	}
}

func (app *application) GetWidgetById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	widgetId, err := strconv.Atoi(id)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	widget, err := app.DB.GetWidget(widgetId)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	out, err := json.MarshalIndent(widget, "", "   ")

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)

}

func (app *application) CreateCustomerAndSubscribeToPlan(w http.ResponseWriter, r *http.Request) {
	var data stripePayload
	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infoLog.Printf("Create Subscription for Email:[%s] , LastFour: [%s] , PaymentMethod: [%s] , Plan: [%s] \n", data.Email, data.LastFour, data.PaymentMethod, data.Plan)

	// build card with secrets
	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: data.Currency,
	}

	okay := true
	var subscription *stripe.Subscription
	txMsg := "Transaction Successful"

	stripeCustomer, msg, err := card.CreateCustomer(data.PaymentMethod, data.Email)

	if err != nil {
		app.errorLog.Println(err)
		okay = false
		txMsg = msg
	}

	if okay {
		subscription, err = card.SubscribeToPlan(stripeCustomer, data.Plan, data.Email, data.LastFour, "")

		if err != nil {
			app.errorLog.Println(err)
			okay = false
			txMsg = msg
		}

		app.infoLog.Printf("Subscription ID -> [{%s}]\n", subscription.ID)
	}

	if okay {
		productID, _ := strconv.Atoi(data.ProductID)
		customerID, err := app.SaveCustomer(data.FirstName, data.LastName, data.Email)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			txMsg = msg
		}

		amount, _ := strconv.Atoi(data.Amount)
		txn := models.Transaction{
			Amount:              amount,
			Currency:            "usd",
			LastFour:            data.LastFour,
			ExpiryMonth:         data.ExpiryMonth,
			ExpiryYear:          data.ExpiryYear,
			TransactionStatusID: 2,
		}
		txnID, err := app.SaveTransaction(txn)

		if err != nil {
			app.errorLog.Println(err)
			return
		}

		order := models.Order{
			WidgetID:      productID,
			TransactionID: txnID,
			CustomerID:    customerID,
			StatusID:      1,
			Quantity:      1,
			Amount:        amount,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		_, err = app.SaveOrder(order)

		if err != nil {
			app.errorLog.Println(err)
			return
		}

	}

	resp := jsonResponse{
		OK:      okay,
		Message: txMsg,
	}

	out, err := json.MarshalIndent(resp, "", "  ")

	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
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
