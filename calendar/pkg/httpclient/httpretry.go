package httpclient

import (
	"bytes"
	"calendar/internal/application/common"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type RetryClient struct {
	delegate   HTTPClient
	maxRetries int
	// Add a predicate function if specific errors should not be retried
	ShouldRetry func(*http.Response, error) bool
	logger      *zap.SugaredLogger
}

func NewRetryClient(delegate HTTPClient, maxRetries int, logger *zap.SugaredLogger) *RetryClient {
	if maxRetries == 0 {
		maxRetries = 3
	}

	return &RetryClient{
		delegate:   delegate,
		maxRetries: maxRetries,
		ShouldRetry: func(resp *http.Response, err error) bool {
			// не ретраим явную отмену/дедлайн
			if err != nil {
				return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
			}
			// resp может быть nil
			if resp == nil {
				return true
			}
			// 5xx и 429 — кандидаты на повтор
			return resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests
		},
		logger: logger,
	}
}

func (c *RetryClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Готовим переиспользуемое тело: если GetBody нет, создаём его один раз
	if req.Body != nil && req.GetBody == nil {
		buf, e := io.ReadAll(req.Body)
		if e != nil {
			return nil, e
		}
		_ = req.Body.Close()
		req.ContentLength = int64(len(buf))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(buf)), nil
		}
		rc, _ := req.GetBody()
		req.Body = rc
	}

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		// Восстановим тело для этой попытки
		if req.GetBody != nil {
			rc, e := req.GetBody()
			if e != nil {
				return nil, e
			}
			req.Body = rc
		}

		// Клон запроса с контекстом
		r := req.Clone(ctx)

		// Шагаем
		resp, err = c.delegate.Do(ctx, r)

		// Условие выхода: успех или дальше ретраить нельзя, либо это последняя попытка
		if !c.ShouldRetry(resp, err) || attempt == c.maxRetries-1 {
			return resp, err
		}

		// Освобождаем соединение в пул перед повтором
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		// Джиттерованный бэкофф
		backoff := common.NextBackoffWithJitter(attempt + 1)
		if backoff <= 0 {
			backoff = 100_000_000 // 100ms, если функция вернула 0
		}

		c.logger.Warnf("retry attempt=%d backoff=%s method=%s url=%s err=%v",
			attempt+1, backoff, req.Method, req.URL.String(), err)

		if err = common.SleepCtx(ctx, backoff); err != nil {
			return resp, fmt.Errorf("retry sleep canceled: %w", err)
		}
	}

	return resp, err
}
