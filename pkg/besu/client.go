package besu

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
)

type Config struct {
	RPCURL          string
	ContractAddress string
	FromAddress     string
	GasLimit        uint64
	ReceiptTimeout  time.Duration
}

type Client struct {
	rpcURL          string
	contractAddress string
	fromAddress     string
	gasLimit        uint64
	receiptTimeout  time.Duration
	httpClient      *http.Client
	now             func() time.Time
}

type CommitResult struct {
	TxHash      string
	BlockNumber int64
	BlockTime   *time.Time
	Mined       bool
}

type LatestRecord struct {
	DataHash  string
	Timestamp *time.Time
	Version   int
	IsRevoked bool
	IsPresent bool
}

type txReceipt struct {
	BlockNumber string `json:"blockNumber"`
}

type blockHeader struct {
	Timestamp string `json:"timestamp"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	ID      int    `json:"id"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewClient(cfg Config) *Client {
	receiptTimeout := cfg.ReceiptTimeout
	if receiptTimeout <= 0 {
		receiptTimeout = 15 * time.Second
	}
	gasLimit := cfg.GasLimit
	if gasLimit == 0 {
		gasLimit = 250000
	}

	return &Client{
		rpcURL:          strings.TrimSpace(cfg.RPCURL),
		contractAddress: strings.TrimSpace(cfg.ContractAddress),
		fromAddress:     strings.TrimSpace(cfg.FromAddress),
		gasLimit:        gasLimit,
		receiptTimeout:  receiptTimeout,
		httpClient:      &http.Client{Timeout: 20 * time.Second},
		now:             time.Now,
	}
}

func (c *Client) CommitHash(ctx context.Context, recordID, dataHash string, timestamp time.Time, version int) (CommitResult, error) {
	input, err := encodeCommitHash(recordID, dataHash, timestamp, version)
	if err != nil {
		return CommitResult{}, err
	}

	var txHash string
	if err := c.rpc(ctx, "eth_sendTransaction", []map[string]string{{
		"from":     c.fromAddress,
		"to":       c.contractAddress,
		"data":     input,
		"gas":      hexUint64(c.gasLimit),
		"gasPrice": "0x0",
	}}, &txHash); err != nil {
		return CommitResult{}, err
	}

	receipt, err := c.waitReceipt(ctx, txHash)
	if err != nil {
		return CommitResult{TxHash: txHash}, err
	}
	if !receipt.Mined {
		return receipt, nil
	}

	blockTime, err := c.getBlockTime(ctx, receipt.BlockNumber)
	if err == nil {
		receipt.BlockTime = blockTime
	}

	return receipt, nil
}

func (c *Client) Verify(ctx context.Context, recordID, dataHash string) (bool, error) {
	input, err := encodeVerify(recordID, dataHash)
	if err != nil {
		return false, err
	}

	var output string
	if err := c.rpc(ctx, "eth_call", []any{
		map[string]string{
			"to":   c.contractAddress,
			"data": input,
		},
		"latest",
	}, &output); err != nil {
		return false, err
	}

	return decodeBool(output)
}

func (c *Client) GetLatest(ctx context.Context, recordID string) (LatestRecord, error) {
	input, err := encodeGetLatest(recordID)
	if err != nil {
		return LatestRecord{}, err
	}

	var output string
	if err := c.rpc(ctx, "eth_call", []any{
		map[string]string{
			"to":   c.contractAddress,
			"data": input,
		},
		"latest",
	}, &output); err != nil {
		return LatestRecord{}, err
	}

	return decodeLatest(output)
}

func (c *Client) Receipt(ctx context.Context, txHash string) (CommitResult, error) {
	var receipt *txReceipt
	if err := c.rpc(ctx, "eth_getTransactionReceipt", []string{txHash}, &receipt); err != nil {
		return CommitResult{}, err
	}
	if receipt == nil {
		return CommitResult{TxHash: txHash, Mined: false}, nil
	}

	blockNumber, err := parseHexInt64(receipt.BlockNumber)
	if err != nil {
		return CommitResult{}, err
	}

	return CommitResult{
		TxHash:      txHash,
		BlockNumber: blockNumber,
		Mined:       true,
	}, nil
}

func (c *Client) waitReceipt(ctx context.Context, txHash string) (CommitResult, error) {
	deadline := c.now().Add(c.receiptTimeout)
	for {
		receipt, err := c.Receipt(ctx, txHash)
		if err != nil {
			return CommitResult{}, err
		}
		if receipt.Mined {
			return receipt, nil
		}
		if c.now().After(deadline) {
			return CommitResult{TxHash: txHash, Mined: false}, nil
		}

		select {
		case <-ctx.Done():
			return CommitResult{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (c *Client) getBlockTime(ctx context.Context, blockNumber int64) (*time.Time, error) {
	var block blockHeader
	if err := c.rpc(ctx, "eth_getBlockByNumber", []any{hexUint64(uint64(blockNumber)), false}, &block); err != nil {
		return nil, err
	}

	seconds, err := parseHexInt64(block.Timestamp)
	if err != nil {
		return nil, err
	}
	timestamp := time.Unix(seconds, 0).UTC()
	return &timestamp, nil
}

func (c *Client) rpc(ctx context.Context, method string, params any, out any) error {
	if c.rpcURL == "" {
		return fmt.Errorf("BESU_RPC_URL is required")
	}
	if c.contractAddress == "" && method != "eth_getTransactionReceipt" && method != "eth_getBlockByNumber" {
		return fmt.Errorf("BESU_CONTRACT_ADDRESS is required")
	}
	if c.fromAddress == "" && method == "eth_sendTransaction" {
		return fmt.Errorf("BESU_FROM_ADDRESS is required")
	}

	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("rpc request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read rpc response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("rpc status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var decoded rpcResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("failed to decode rpc response: %w", err)
	}
	if decoded.Error != nil {
		return fmt.Errorf("rpc error %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(decoded.Result, out); err != nil {
		return fmt.Errorf("failed to decode rpc result: %w", err)
	}

	return nil
}

func encodeCommitHash(recordID, dataHash string, timestamp time.Time, version int) (string, error) {
	dataWord, err := hashWord(dataHash)
	if err != nil {
		return "", err
	}
	return encodeMethod(
		"commitHash(string,bytes32,uint256,uint256)",
		encodeDynamicString(recordID),
		dataWord,
		encodeUint256(uint64(timestamp.UTC().Unix())),
		encodeUint256(uint64(version)),
	), nil
}

func encodeGetLatest(recordID string) (string, error) {
	return encodeMethod("getLatest(string)", encodeDynamicString(recordID)), nil
}

func encodeVerify(recordID, dataHash string) (string, error) {
	dataWord, err := hashWord(dataHash)
	if err != nil {
		return "", err
	}
	return encodeMethod("verify(string,bytes32)", encodeDynamicString(recordID), dataWord), nil
}

func encodeMethod(signature string, args ...[]byte) string {
	selector := methodSelector(signature)
	head := make([]byte, 0, 32*len(args))
	tail := make([]byte, 0)
	offset := 32 * len(args)
	for _, arg := range args {
		if len(arg) == 32 {
			head = append(head, arg...)
			continue
		}
		head = append(head, encodeUint256(uint64(offset))...)
		tail = append(tail, arg...)
		offset += len(arg)
	}

	return "0x" + hex.EncodeToString(append(selector, append(head, tail...)...))
}

func encodeDynamicString(value string) []byte {
	data := []byte(value)
	paddedLen := ((len(data) + 31) / 32) * 32
	out := make([]byte, 0, 32+paddedLen)
	out = append(out, encodeUint256(uint64(len(data)))...)
	out = append(out, data...)
	if paddedLen > len(data) {
		out = append(out, make([]byte, paddedLen-len(data))...)
	}
	return out
}

func encodeUint256(value uint64) []byte {
	word := make([]byte, 32)
	for i := 0; i < 8; i++ {
		word[31-i] = byte(value >> (8 * i))
	}
	return word
}

func methodSelector(signature string) []byte {
	hasher := sha3.NewLegacyKeccak256()
	_, _ = hasher.Write([]byte(signature))
	return hasher.Sum(nil)[:4]
}

func hashWord(value string) ([]byte, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if len(trimmed) != 64 {
		return nil, fmt.Errorf("data hash must be 32 bytes encoded as 64 hex chars")
	}
	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid data hash: %w", err)
	}
	return decoded, nil
}

func decodeBool(output string) (bool, error) {
	words, err := decodeWords(output)
	if err != nil {
		return false, err
	}
	if len(words) < 1 {
		return false, fmt.Errorf("missing bool return value")
	}
	return words[0][31] == 1, nil
}

func decodeLatest(output string) (LatestRecord, error) {
	words, err := decodeWords(output)
	if err != nil {
		return LatestRecord{}, err
	}
	if len(words) < 4 {
		return LatestRecord{}, fmt.Errorf("expected 4 return words, got %d", len(words))
	}

	dataHash := hex.EncodeToString(words[0])
	timestampValue, err := parseWordUint(words[1])
	if err != nil {
		return LatestRecord{}, err
	}
	versionValue, err := parseWordUint(words[2])
	if err != nil {
		return LatestRecord{}, err
	}
	isRevoked := words[3][31] == 1
	if isZeroWord(words[0]) && timestampValue == 0 && versionValue == 0 && !isRevoked {
		return LatestRecord{}, nil
	}

	record := LatestRecord{
		DataHash:  dataHash,
		Version:   int(versionValue),
		IsRevoked: isRevoked,
		IsPresent: true,
	}
	if timestampValue > 0 {
		timestamp := time.Unix(int64(timestampValue), 0).UTC()
		record.Timestamp = &timestamp
	}

	return record, nil
}

func decodeWords(output string) ([][]byte, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(output), "0x")
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed)%64 != 0 {
		return nil, fmt.Errorf("invalid abi output length")
	}

	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to decode abi output: %w", err)
	}

	words := make([][]byte, 0, len(decoded)/32)
	for i := 0; i < len(decoded); i += 32 {
		words = append(words, decoded[i:i+32])
	}
	return words, nil
}

func parseWordUint(word []byte) (uint64, error) {
	if len(word) != 32 {
		return 0, fmt.Errorf("invalid word length")
	}
	var value uint64
	for _, b := range word[24:] {
		value = (value << 8) | uint64(b)
	}
	return value, nil
}

func parseHexInt64(raw string) (int64, error) {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "0x"))
	if value == "" {
		return 0, fmt.Errorf("missing hex value")
	}
	parsed, err := strconv.ParseInt(value, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse hex int: %w", err)
	}
	return parsed, nil
}

func hexUint64(value uint64) string {
	return "0x" + strconv.FormatUint(value, 16)
}

func isZeroWord(word []byte) bool {
	for _, b := range word {
		if b != 0 {
			return false
		}
	}
	return true
}
