package model

type PaymentStatus string
type PaymentMethod string
type PaymentGatewayProvider string
type PaymentEntryType string
type FeeScope string           // 👈 TAMBAHAN
type GatewayEventStatus string // (opsional, buat payment_gateway_events)

const (
	PaymentStatusInitiated         PaymentStatus = "initiated"
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusAwaitingCallback  PaymentStatus = "awaiting_callback"
	PaymentStatusPaid              PaymentStatus = "paid"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
	PaymentStatusRefunded          PaymentStatus = "refunded"
	PaymentStatusFailed            PaymentStatus = "failed"
	PaymentStatusCanceled          PaymentStatus = "canceled"
	PaymentStatusExpired           PaymentStatus = "expired"
)

const (
	PaymentMethodGateway      PaymentMethod = "gateway"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodQRIS         PaymentMethod = "qris"
	PaymentMethodOther        PaymentMethod = "other"
)

const (
	GatewayProviderMidtrans PaymentGatewayProvider = "midtrans"
	GatewayProviderXendit   PaymentGatewayProvider = "xendit"
	GatewayProviderTripay   PaymentGatewayProvider = "tripay"
	GatewayProviderDuitku   PaymentGatewayProvider = "duitku"
	GatewayProviderNicepay  PaymentGatewayProvider = "nicepay"
	GatewayProviderStripe   PaymentGatewayProvider = "stripe"
	GatewayProviderPaypal   PaymentGatewayProvider = "paypal"
	GatewayProviderOther    PaymentGatewayProvider = "other"
)

const (
	PaymentEntryCharge     PaymentEntryType = "charge"
	PaymentEntryPayment    PaymentEntryType = "payment"
	PaymentEntryRefund     PaymentEntryType = "refund"
	PaymentEntryAdjustment PaymentEntryType = "adjustment"
)

// ===== enum fee_scope (mirror DB) =====
const (
	FeeScopeTenant      FeeScope = "tenant"
	FeeScopeClassParent FeeScope = "class_parent"
	FeeScopeClass       FeeScope = "class"
	FeeScopeSection     FeeScope = "section"
	FeeScopeStudent     FeeScope = "student"
	FeeScopeTerm        FeeScope = "term"
)

// (opsional) kalau kamu pakai gateway_event_status di model webhook
const (
	GatewayEventStatusReceived   GatewayEventStatus = "received"
	GatewayEventStatusProcessing GatewayEventStatus = "processing"
	GatewayEventStatusSuccess    GatewayEventStatus = "success"
	GatewayEventStatusFailed     GatewayEventStatus = "failed"
)
