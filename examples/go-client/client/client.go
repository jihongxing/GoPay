package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client GoPay 客户端
type Client struct {
	BaseURL    string
	AppID      string
	HTTPClient *http.Client
}

// NewClient 创建新的客户端
func NewClient(baseURL, appID string) *Client {
	return &Client{
		BaseURL: baseURL,
		AppID:   appID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateOrder 创建支付订单
func (c *Client) CreateOrder(req *CreateOrderRequest) (*CreateOrderResponse, error) {
	// 设置 app_id
	req.AppID = c.AppID

	// 发送请求
	resp, err := c.post("/api/v1/checkout", req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// 解析响应
	var result struct {
		Code    int                   `json:"code"`
		Message string                `json:"message"`
		Data    *CreateOrderResponse  `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("create order failed: %s", result.Message)
	}

	return result.Data, nil
}

// QueryOrder 查询订单状态
func (c *Client) QueryOrder(orderNo string) (*Order, error) {
	// 发送请求
	resp, err := c.get(fmt.Sprintf("/api/v1/orders/%s", orderNo))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// 解析响应
	var result struct {
		Code    int     `json:"code"`
		Message string  `json:"message"`
		Data    *Order  `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("query order failed: %s", result.Message)
	}

	return result.Data, nil
}

// post 发送 POST 请求
func (c *Client) post(path string, body interface{}) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// get 发送 GET 请求
func (c *Client) get(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
