package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Donation struct {
	DonationID             uint           `gorm:"column:donation_id;primaryKey" json:"donation_id"`                                                    // ID unik donasi
	DonationUserID         *uuid.UUID     `gorm:"column:donation_user_id;type:uuid" json:"donation_user_id,omitempty"`                                 // ID user yang berdonasi
	DonationAmount         int            `gorm:"column:donation_amount;not null" json:"donation_amount"`                                              // Jumlah donasi (wajib)
	DonationMessage        string         `gorm:"column:donation_message;type:text" json:"donation_message"`                                           // Pesan/ucapan (opsional)
	DonationStatus         string         `gorm:"column:donation_status;type:varchar(20);default:'pending'" json:"donation_status"`                    // pending, paid, expired, etc.
	DonationOrderID        string         `gorm:"column:donation_order_id;type:varchar(100);unique;not null" json:"donation_order_id"`                 // Order ID unik
	DonationPaymentToken   string         `gorm:"column:donation_payment_token;type:text" json:"donation_payment_token"`                               // Token Snap/gateway
	DonationPaymentGateway string         `gorm:"column:donation_payment_gateway;type:varchar(50);default:'midtrans'" json:"donation_payment_gateway"` // Nama gateway
	DonationPaymentMethod  string         `gorm:"column:donation_payment_method;type:varchar(50)" json:"donation_payment_method"`                      // Metode spesifik (gopay, va_bni)
	DonationPaidAt         *time.Time     `gorm:"column:donation_paid_at" json:"donation_paid_at"`                                                     // Waktu pembayaran sukses (nullable)
	CreatedAt              time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

// TableName override nama tabel
func (Donation) TableName() string {
	return "donations"
}
