package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RPCClient represents an Ethereum JSON-RPC client
type RPCClient struct {
	URL     string
	client  *http.Client
	timeout time.Duration
}

// NewRPCClient creates a new RPC client with the given URL
func NewRPCClient(url string) *RPCClient {
	return &RPCClient{
		URL:     url,
		client:  &http.Client{},
		timeout: 30 * time.Second,
	}
}

// SetTimeout sets the timeout for RPC requests
func (r *RPCClient) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
	r.client.Timeout = timeout
}

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TransactionReceipt represents an Ethereum transaction receipt
type TransactionReceipt struct {
	TransactionHash   string `json:"transactionHash"`
	TransactionIndex  string `json:"transactionIndex"`
	BlockHash         string `json:"blockHash"`
	BlockNumber       string `json:"blockNumber"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	GasUsed           string `json:"gasUsed"`
	ContractAddress   string `json:"contractAddress"`
	Status            string `json:"status"`
	From              string `json:"from"`
	To                string `json:"to"`
}

// Call makes a JSON-RPC call
func (r *RPCClient) Call(method string, params []interface{}) (*JSONRPCResponse, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", r.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: r.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var response JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", response.Error.Code, response.Error.Message)
	}

	return &response, nil
}

// GetTransactionReceipt gets the transaction receipt for a given hash
func (r *RPCClient) GetTransactionReceipt(txHash string) (*TransactionReceipt, error) {
	response, err := r.Call("eth_getTransactionReceipt", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	if response.Result == nil {
		return nil, fmt.Errorf("transaction not found or not yet mined")
	}

	receiptData, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal receipt data: %w", err)
	}

	var receipt TransactionReceipt
	if err := json.Unmarshal(receiptData, &receipt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
	}

	return &receipt, nil
}

// VerifyTransactionSuccess verifies that a transaction was successful
func (r *RPCClient) VerifyTransactionSuccess(txHash string) (bool, *TransactionReceipt, error) {
	receipt, err := r.GetTransactionReceipt(txHash)
	if err != nil {
		return false, nil, err
	}

	// Status "0x1" means success, "0x0" means failure
	success := receipt.Status == "0x1"
	return success, receipt, nil
}

// GetBlockNumber gets the current block number
func (r *RPCClient) GetBlockNumber() (string, error) {
	response, err := r.Call("eth_blockNumber", []interface{}{})
	if err != nil {
		return "", err
	}

	if response.Result == nil {
		return "", fmt.Errorf("no block number returned")
	}

	blockNumber, ok := response.Result.(string)
	if !ok {
		return "", fmt.Errorf("invalid block number format")
	}

	return blockNumber, nil
}
