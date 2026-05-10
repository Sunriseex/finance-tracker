package dto

type CreateTransferRequest struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	AmountMinor   int64  `json:"amount_minor"`
	Description   string `json:"description"`
}

type TransferResponse struct {
	Out          TransactionResponse `json:"out"`
	In           TransactionResponse `json:"in"`
	ExchangeRate string              `json:"exchange_rate"`
}
