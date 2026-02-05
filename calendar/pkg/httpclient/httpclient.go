package httpclient

import (
	"calendar/pkg/config"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

type HTTPClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

type Client struct {
	http *http.Client
	tr   *http.Transport
	ua   string
}

func NewClient(cfg config.HTTPClient) *Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		DisableKeepAlives:     !cfg.KeepAlives,
		ForceAttemptHTTP2:     true,
	}

	// Настройка TLS: отключение проверки SSL сертификатов, если указано в конфиге
	if cfg.InsecureSkipVerify {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	cl := &http.Client{Transport: tr, Timeout: cfg.ClientTimeout}

	return &Client{http: cl, tr: tr, ua: cfg.UserAgent}
}

func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req = req.WithContext(ctx)
	if c.ua != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.ua)
	}
	return c.http.Do(req)
}

func (c *Client) CloseIdle()                 { c.tr.CloseIdleConnections() }
func (c *Client) Transport() *http.Transport { return c.tr }
