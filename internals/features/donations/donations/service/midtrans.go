package service

import (
	"masjidku_backend/internals/features/donations/donations/model"

	midtrans "github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

var SnapClient snap.Client

// InitMidtrans menginisialisasi Midtrans Snap Client dengan server key.
func InitMidtrans(serverKey string) {
	SnapClient.New(serverKey, midtrans.Sandbox)
}

// GenerateSnapToken membuat token Snap Midtrans berdasarkan data donasi dan user.
func GenerateSnapToken(d model.Donation, name string, email string) (string, error) {
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
		return "", err
	}

	return resp.Token, nil
}
