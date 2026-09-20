package kraken

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Khuako/usdt-rate-service/internal/rates"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
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

var ErrRequest = errors.New("kraken http status")

func (c *Client) GetOrderBook(ctx context.Context) (rates.OrderBook, error) {
	ctx, span := otel.Tracer("usdt-rate-service/internal/kraken").Start(ctx, "kraken.GetOrderbook")
	defer span.End()
	res, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam(
			"pair",
			"USDTUSD").SetQueryParam(
		"count",
		"100",
	).Get("/0/public/Depth")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch order book")
		return rates.OrderBook{}, err
	}
	if !res.IsSuccess() {
		span.RecordError(fmt.Errorf("%w: %d", ErrRequest, res.StatusCode()))
		span.SetStatus(codes.Error, "failed to fetch order book")
		return rates.OrderBook{}, fmt.Errorf("%w: %d", ErrRequest, res.StatusCode())
	}
	curTime := time.Now().UTC()
	orderBook, parseError := parseOrderBook(res.Body())
	if parseError != nil {
		span.RecordError(parseError)
		span.SetStatus(codes.Error, "failed to parse order book")
		return orderBook, fmt.Errorf("parsing orderBook: %w", parseError)
	}
	orderBook.ReceivedAt = curTime
	return orderBook, nil
}
