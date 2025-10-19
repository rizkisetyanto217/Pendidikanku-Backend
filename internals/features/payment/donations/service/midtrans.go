package service

import (
	"masjidku_backend/internals/features/payment/donations/model"

	midtrans "github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

var SnapClient snap.Client

// Panggil saat bootstrap app (sandbox)
func InitMidtrans(serverKey string) {
	SnapClient.New(serverKey, midtrans.Sandbox)
}

// Buat Snap token + redirect_url
func GenerateSnapToken(d model.Donation, name, email string) (string, string, error) {
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  d.DonationOrderID,
			GrossAmt: int64(d.DonationAmount),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: name,
			Email: email,
		},
	}

	resp, err := SnapClient.CreateTransaction(req)
	if err != nil {
		return "", "", err
	}

	return resp.Token, resp.RedirectURL, nil
}
