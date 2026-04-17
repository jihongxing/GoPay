package client

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Client GoPay 客户端
type Client struct {
	BaseURL    string
	AppID      string
	AppSecret  string
	HTTPClient *http.Client
}

// NewClient 创建新的客户端
func NewClient(baseURL, appID, appSecret string) *Client {
	return &Client{
		BaseURL:   baseURL,
		AppID:     appID,
		AppSecret: appSecret,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIResponse 统一响应格式
type APIResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// CreateOrder 创建支付订单
func (c *Client) CreateOrder(req *CreateOrderRequest) (*CreateOrderResponse, error) {
	req.AppID = c.AppID

	resp, err := c.post("/api/v1/checkout", req)
	if err != nil {
		return nil, err
	}

	var result CreateOrderResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("parse response data failed: %w", err)
	}
	return &result, nil
}

// QueryOrder 查询订单状态
func (c *Client) QueryOrder(orderNo string) (*Order, error) {
	resp, err := c.get(fmt.Sprintf("/api/v1/orders/%s", orderNo))
	if err != nil {
		return nil, err
	}

	var result Order
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("parse response data failed: %w", err)
	}
	return &result, nil
}

// sign 生成签名头
func (c *Client) sign(body []byte) (timestamp, nonce, signature string) {
	timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	nonce = uuid.New().String()

	message := string(body) + "\n" + timestamp + "\n" + nonce
	mac := hmac.New(sha256.New, []byte(c.AppSecret))
	mac.Write([]byte(message))
	signature = hex.EncodeToString(mac.Sum(nil))
	return
}

// post 发送 POST 请求（带签名）
func (c *Client) post(path string, body interface{}) (*APIResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	ts, nonce, sig := c.sign(data)
	req.Header.Set("X-App-ID", c.AppID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", sig)

	return c.doRequest(req)
}

// get 发送 GET 请求（带签名）
func (c *Client) get(path string) (*APIResponse, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// GET 请求 body 为空
	ts, nonce, sig := c.sign([]byte(""))
	req.Header.Set("X-App-ID", c.AppID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", sig)

	return c.doRequest(req)
}

// doRequest 执行请求并解析响应
func (c *Client) doRequest(req *http.Request) (*APIResponse, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	if apiResp.Code != "SUCCESS" {
		return nil, fmt.Errorf("[%s] %s", apiResp.Code, apiResp.Message)
	}

	return &apiResp, nil
}
