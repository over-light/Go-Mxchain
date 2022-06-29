package common

// GetProofResponse is a struct that stores the response of a GetProof API request
type GetProofResponse struct {
	Proof    [][]byte
	Value    []byte
	RootHash string
}

// TransactionsPoolAPIResponse is a struct that holds the data to be returned when getting the transaction pool from an API call
type TransactionsPoolAPIResponse struct {
	RegularTransactions  []string `json:"regularTransactions"`
	SmartContractResults []string `json:"smartContractResults"`
	Rewards              []string `json:"rewards"`
}

// Transaction is a struct that holds transaction data to be returned when getting the transactions from pool
type Transaction struct {
	Hash     string `json:"hash,omitempty"`
	Nonce    uint64 `json:"nonce,omitempty"`
	Sender   string `json:"sender,omitempty"`
	Receiver string `json:"receiver,omitempty"`
	GasLimit uint64 `json:"gasLimit,omitempty"`
	GasPrice uint64 `json:"gasPrice,omitempty"`
}

// TransactionsPoolForSenderApiResponse is a struct that holds the data to be returned when getting the transactions for a sender from an API call
type TransactionsPoolForSenderApiResponse struct {
	Transactions []Transaction `json:"transactions"`
}

// NonceGapApiResponse is a struct that holds a nonce gap from transactions pool
// From - last known nonce
// To   - next known nonce
type NonceGapApiResponse struct {
	From uint64 `json:"from"`
	To   uint64 `json:"to"`
}

// TransactionsPoolNonceGapsForSenderApiResponse is a struct that holds the data to be returned when getting the nonce gaps from transactions pool for a sender from an API call
type TransactionsPoolNonceGapsForSenderApiResponse struct {
	Sender string                `json:"sender"`
	Gaps   []NonceGapApiResponse `json:"gaps"`
}
