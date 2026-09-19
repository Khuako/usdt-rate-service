package kraken

import (
	"context"
	"fmt"
	"time"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/go-resty/resty/v2"
)

type Client struct {
	httpClient *resty.Client
}

func NewClient(baseUrl string) *Client {
	c := resty.New()
	c.BaseURL = baseUrl
	c.SetTimeout(time.Second * 5)
	return &Client{c}
}

func (c *Client) GetOrderBook(ctx context.Context) (rates.OrderBook, error) {

	res, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam(
			"pair",
			"USDTUSD").SetQueryParam(
		"count",
		"100",
	).Get("/0/public/Depth")
	if err != nil {
		return rates.OrderBook{}, err
	}
	if !res.IsSuccess() {
		return rates.OrderBook{}, fmt.Errorf("kraken HTTP status: %d", res.StatusCode())
	}
	curTime := time.Now().UTC()
	orderBook, parseError := parseOrderBook(res.Body())
	if parseError != nil {
		return orderBook, fmt.Errorf("parsing orderBook: %w", parseError)
	}
	orderBook.ReceivedAt = curTime
	return orderBook, nil
}
