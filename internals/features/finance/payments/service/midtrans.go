package service

import (
	"errors"
	"time"

	"schoolku_backend/internals/features/finance/payments/model"

	midtrans "github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

/* =========================================================
   Midtrans Client
========================================================= */

var SnapClient snap.Client

// InitMidtrans harus dipanggil saat bootstrap app.
// useProduction=true untuk Production, false untuk Sandbox.
func InitMidtrans(serverKey string, useProduction bool) {
	if useProduction {
		SnapClient.New(serverKey, midtrans.Production)
	} else {
		SnapClient.New(serverKey, midtrans.Sandbox)
	}
}

/* =========================================================
   Input helper untuk data customer
========================================================= */

type CustomerInput struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
	Address   string // optional
	City      string // optional
	Postcode  string // optional
	Country   string // optional, default "IDN"
}

/*
	=========================================================
	  Generate Snap Token

=========================================================
*/
func GenerateSnapToken(p model.Payment, cust CustomerInput) (string, string, error) {
	if p.PaymentAmountIDR <= 0 {
		return "", "", errors.New("invalid payment_amount_idr")
	}
	if p.PaymentExternalID == nil || *p.PaymentExternalID == "" {
		return "", "", errors.New("payment_external_id is required (used as OrderID)")
	}

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  *p.PaymentExternalID,
			GrossAmt: int64(p.PaymentAmountIDR),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: cust.FirstName,
			LName: cust.LastName,
			Email: cust.Email,
			Phone: cust.Phone,
			BillAddr: &midtrans.CustomerAddress{
				FName:       cust.FirstName,
				LName:       cust.LastName,
				Phone:       cust.Phone,
				Address:     cust.Address,
				City:        cust.City,
				Postcode:    cust.Postcode,
				CountryCode: defaultString(cust.Country, "IDN"),
			},
			ShipAddr: &midtrans.CustomerAddress{
				FName:       cust.FirstName,
				LName:       cust.LastName,
				Phone:       cust.Phone,
				Address:     cust.Address,
				City:        cust.City,
				Postcode:    cust.Postcode,
				CountryCode: defaultString(cust.Country, "IDN"),
			},
		},
	}

	if p.PaymentDescription != nil && *p.PaymentDescription != "" {
		req.CreditCard = &snap.CreditCardDetails{Secure: true}
		req.CustomField1 = truncate(*p.PaymentDescription, 40)
	}

	req.Items = &[]midtrans.ItemDetails{
		{
			ID:       safe(*p.PaymentExternalID),
			Price:    int64(p.PaymentAmountIDR),
			Qty:      1,
			Name:     firstNonEmpty(p.PaymentDescription, stringPtr("SPP Payment")),
			Category: "SPP",
		},
	}

	resp, err := SnapClient.CreateTransaction(req)
	if err != nil {
		return "", "", err
	}
	return resp.Token, resp.RedirectURL, nil
}

/* =========================================================
   Utils
========================================================= */

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

func defaultString(s string, def string) string {
	if s == "" {
		return def
	}
	return s
}

func firstNonEmpty(ps *string, def *string) string {
	if ps != nil && *ps != "" {
		return *ps
	}
	if def != nil {
		return *def
	}
	return ""
}

func stringPtr(s string) *string { return &s }

func safe(s string) string {
	if s == "" {
		return "item-1"
	}
	return s
}

func minutesUntil(target time.Time, now time.Time) int64 {
	d := target.Sub(now)
	if d <= 0 {
		return 0
	}
	return int64(d.Round(time.Minute) / time.Minute)
}
