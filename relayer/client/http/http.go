package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
)

// RetryableClientConfig is the config for the RetryableClient.
type RetryableClientConfig struct {
	RetriesCount   int
	RequestTimeout time.Duration
	DoTimeout      time.Duration
	RetryDelay     time.Duration
}

// DefaultClientConfig returns default RetryableClientConfig.
func DefaultClientConfig() RetryableClientConfig {
	return RetryableClientConfig{
		RetriesCount:   5,
		RequestTimeout: 10 * time.Second,
		DoTimeout:      30 * time.Second,
		RetryDelay:     300 * time.Millisecond,
	}
}

// RetryableClient is HTTP RetryableClient.
type RetryableClient struct {
	cfg RetryableClientConfig
}

// NewRetryableClient returns new instance RetryableClient.
func NewRetryableClient(cfg RetryableClientConfig) RetryableClient {
	return RetryableClient{
		cfg: cfg,
	}
}

// DoJSON executes the HTTP application/json request with retires based on the client configuration.
func (c RetryableClient) DoJSON(ctx context.Context, method, url string, reqBody any, resDecoder func([]byte) error) error {
	doCtx, doCtxCancel := context.WithTimeout(ctx, c.cfg.DoTimeout)
	defer doCtxCancel()
	return retry.Do(doCtx, c.cfg.RetryDelay, func() error {
		reqCtx, reqCtxCancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
		defer reqCtxCancel()

		return doJSON(reqCtx, method, url, reqBody, resDecoder)
	})
}

func doJSON(ctx context.Context, method, url string, reqBody any, resDecoder func([]byte) error) error {
	var reqBodyReader io.Reader
	if reqBody != nil {
		reqBodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return errors.Errorf("can't marshal request body, err: %v", err)
		}
		reqBodyReader = bytes.NewReader(reqBodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBodyReader)
	if err != nil {
		return errors.Errorf("can't build the request, err: %v", err)
	}

	// fix for the EOF error
	req.Close = true
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Errorf("can't perform the request, err: %v", err)
	}

	defer resp.Body.Close()
	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf("can't read the response body, err: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.Errorf("can't perform request, code: %d, body: %s", resp.StatusCode, string(bodyData))
	}

	err = resDecoder(bodyData)
	if err != nil {
		return errors.Errorf("can't docde the response body, body: %s, err: %v", string(bodyData), err)
	}

	return nil
}
