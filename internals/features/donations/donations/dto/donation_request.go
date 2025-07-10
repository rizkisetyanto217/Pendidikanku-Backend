package dto

type CreateDonationRequest struct {
	UserID  string `json:"user_id"`
	Amount  int    `json:"amount"`
	Message string `json:"message"`
	Name    string `json:"name"`
	Email   string `json:"email"`
}

