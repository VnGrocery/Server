package besu

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

type Config struct {
	RPCURL          string
	RPCURLs         []string
	ContractAddress string
	FromAddress     string
	PrivateKey      string
	ChainID         string
	GasLimit        uint64
	ReceiptTimeout  time.Duration
}

type Client struct {
	rpcURLs         []string
	contractAddress string
	fromAddress     string
	privateKey      *ecdsa.PrivateKey
	chainID         *big.Int
	gasLimit        uint64
	receiptTimeout  time.Duration
	httpClient      *http.Client
	now             func() time.Time
	nextEndpoint    atomic.Uint64
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

	privateKey := mustParsePrivateKey(cfg.PrivateKey)
	fromAddress := strings.TrimSpace(cfg.FromAddress)
	if fromAddress == "" && privateKey != nil {
		fromAddress = crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	}

	rpcURLs := normalizeRPCURLs(append([]string{cfg.RPCURL}, cfg.RPCURLs...))

	return &Client{
		rpcURLs:         rpcURLs,
		contractAddress: strings.TrimSpace(cfg.ContractAddress),
		fromAddress:     fromAddress,
		privateKey:      privateKey,
		chainID:         parseChainID(cfg.ChainID),
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
	if c.privateKey != nil {
		txHash, err = c.sendRawTransaction(ctx, input)
	} else {
		err = c.rpc(ctx, "eth_sendTransaction", []map[string]string{{
			"from":     c.fromAddress,
			"to":       c.contractAddress,
			"data":     input,
			"gas":      hexUint64(c.gasLimit),
			"gasPrice": "0x0",
		}}, &txHash)
		if err != nil && strings.Contains(err.Error(), "The method eth_sendTransaction is not supported") {
			return CommitResult{}, fmt.Errorf("besu requires signed raw transaction; set BESU_PRIVATE_KEY to enable eth_sendRawTransaction: %w", err)
		}
	}
	if err != nil {
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

func (c *Client) RevokeHash(ctx context.Context, recordID string, version int) (CommitResult, error) {
	input := encodeMethod(
		"revokeHash(string,uint256)",
		encodeDynamicString(recordID),
		encodeUint256(uint64(version)),
	)

	var txHash string
	var err error
	if c.privateKey != nil {
		txHash, err = c.sendRawTransaction(ctx, input)
	} else {
		err = c.rpc(ctx, "eth_sendTransaction", []map[string]string{{
			"from":     c.fromAddress,
			"to":       c.contractAddress,
			"data":     input,
			"gas":      hexUint64(c.gasLimit),
			"gasPrice": "0x0",
		}}, &txHash)
		if err != nil && strings.Contains(err.Error(), "The method eth_sendTransaction is not supported") {
			return CommitResult{}, fmt.Errorf("besu requires signed raw transaction; set BESU_PRIVATE_KEY to enable eth_sendRawTransaction: %w", err)
		}
	}
	if err != nil {
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
	if len(c.rpcURLs) == 0 {
		return fmt.Errorf("BESU_RPC_URL or BESU_RPC_URLS is required")
	}
	if c.contractAddress == "" && method != "eth_getTransactionReceipt" && method != "eth_getBlockByNumber" {
		return fmt.Errorf("BESU_CONTRACT_ADDRESS is required")
	}
	if c.fromAddress == "" && (method == "eth_sendTransaction" || method == "eth_getTransactionCount") {
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

	start := int(c.nextEndpoint.Add(1)-1) % len(c.rpcURLs)
	errorsByEndpoint := make([]string, 0, len(c.rpcURLs))
	for i := 0; i < len(c.rpcURLs); i++ {
		index := (start + i) % len(c.rpcURLs)
		url := c.rpcURLs[index]
		if err := c.callRPC(ctx, url, body, out); err != nil {
			errorsByEndpoint = append(errorsByEndpoint, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		c.nextEndpoint.Store(uint64(index + 1))
		return nil
	}
	return fmt.Errorf("all Besu RPC endpoints failed: %s", strings.Join(errorsByEndpoint, "; "))
}

func (c *Client) callRPC(ctx context.Context, rpcURL string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded rpcResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if decoded.Error != nil {
		return fmt.Errorf("rpc error %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(decoded.Result, out); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}
	return nil
}

func normalizeRPCURLs(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}
	return result
}

func (c *Client) sendRawTransaction(ctx context.Context, input string) (string, error) {
	nonce, err := c.pendingNonce(ctx)
	if err != nil {
		return "", err
	}

	to := common.HexToAddress(c.contractAddress)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Gas:      c.gasLimit,
		GasPrice: big.NewInt(0),
		Value:    big.NewInt(0),
		Data:     common.FromHex(input),
	})

	chainID := c.chainID
	if chainID == nil || chainID.Sign() <= 0 {
		chainID = big.NewInt(1337)
	}

	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to encode signed transaction: %w", err)
	}

	var txHash string
	if err := c.rpc(ctx, "eth_sendRawTransaction", []string{hexutil.Encode(raw)}, &txHash); err != nil {
		return "", err
	}
	return txHash, nil
}

func (c *Client) pendingNonce(ctx context.Context) (uint64, error) {
	var raw string
	if err := c.rpc(ctx, "eth_getTransactionCount", []string{c.fromAddress, "pending"}, &raw); err != nil {
		return 0, err
	}
	value := strings.TrimSpace(strings.TrimPrefix(raw, "0x"))
	if value == "" {
		return 0, nil
	}
	nonce, err := strconv.ParseUint(value, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse nonce: %w", err)
	}
	return nonce, nil
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

func mustParsePrivateKey(raw string) *ecdsa.PrivateKey {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "0x"))
	if value == "" {
		return nil
	}
	key, err := crypto.HexToECDSA(value)
	if err != nil {
		return nil
	}
	return key
}

func parseChainID(raw string) *big.Int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil
	}
	return parsed
}
