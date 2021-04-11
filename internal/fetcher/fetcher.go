package fetcher

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"crawler/internal/ctxutils"
	"crawler/internal/logger"
	"crawler/internal/models"
)

var log = logger.G().Named("fetcher").Sugar()

type Interface interface {
	Fetch(ctx context.Context, url string) ([]*models.Product, error)
}

type client struct{ http *http.Client }

func New() (Interface, error) {
	return &client{
		http: &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
			Timeout:   30 * time.Second,
		},
	}, nil
}

func (c *client) Fetch(ctx context.Context, url string) ([]*models.Product, error) {
	resource, err := c.getResource(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("get resource failed: %w", err)
	}
	defer resource.Close()

	list, err := c.parseResource(resource, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("parse resourece failed: %w", err)
	}
	return list, err
}

func (c *client) parseResource(resource io.Reader, fetchTimestamp int64) ([]*models.Product, error) {
	dict := make(map[string]*models.Product, 100)
	reader := csv.NewReader(resource)
	for {
		line, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read failed: %w", err)
		}

		if len(line) != 2 {
			return nil, fmt.Errorf("unexpected len of line - %d", len(line))
		}
		name := line[0]
		price, err := strconv.ParseUint(line[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price value - %s", line[1])
		}

		product, exists := dict[name]
		if exists {
			product.Price = uint32(price)
			product.PriceChangesCounter++
		} else {
			dict[name] = &models.Product{
				Name:         name,
				Price:        uint32(price),
				LastUpdateTs: uint64(fetchTimestamp),
			}
		}
	}

	list := make([]*models.Product, 0, len(dict))
	for _, value := range dict {
		list = append(list, value)
	}
	return list, nil
}

func (c *client) getResource(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		drainResponse(resp)
		return nil, fmt.Errorf("error status code %d", resp.StatusCode)
	}
	log.Infow("resource", "request-id", ctxutils.GetRequestID(ctx), "content-length", resp.ContentLength)
	return resp.Body, nil
}

func drainResponse(resp *http.Response) {
	_, _ = io.Copy(ioutil.Discard, resp.Body)
	_ = resp.Body.Close()
}
