package fluzweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36"

var ErrChallenge = errors.New("fluz session invalid or cloudflare challenge; re-login required")

type Config struct {
	Cookie      string
	PIN         string
	Endpoint    string
	ProxyURL    string
	SessionPath string
	Relogin     func() (string, error)
}

type Client struct {
	sess     *Session
	pin      string
	relogin  func() (string, error)
	http     *http.Client
	endpoint string
}

func New(cfg Config) *Client {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	transport := &http.Transport{}
	if cfg.ProxyURL != "" {
		if u, err := url.Parse(cfg.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}

	var sess *Session
	if cfg.SessionPath != "" {
		if s, err := LoadSession(cfg.SessionPath); err == nil && s.Header() != "" {
			sess = s
		}
	}
	if sess == nil {
		sess = NewSession(cfg.Cookie, cfg.SessionPath)
	}

	return &Client{
		sess:     sess,
		pin:      cfg.PIN,
		relogin:  cfg.Relogin,
		http:     &http.Client{Transport: transport, Timeout: 45 * time.Second},
		endpoint: endpoint,
	}
}

func (c *Client) Cookie() string { return c.sess.Header() }

func (c *Client) Heartbeat() error {
	status, raw, err := c.doRequest(http.MethodGet, "/session/heartbeat", nil)
	if err != nil {
		return err
	}
	if isChallenge(status, raw) {
		return c.tryRelogin()
	}
	return nil
}

func (c *Client) KeepAlive(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = c.Heartbeat()
		}
	}
}

func (c *Client) post(path string, body any) ([]any, error) {
	stream, err := c.postOnce(path, body)
	if errors.Is(err, ErrChallenge) {
		if rerr := c.tryRelogin(); rerr != nil {
			return nil, rerr
		}
		return c.postOnce(path, body)
	}
	return stream, err
}

func (c *Client) postOnce(path string, body any) ([]any, error) {
	status, raw, err := c.doRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	var stream []any
	if err := json.Unmarshal(raw, &stream); err != nil {
		if isChallenge(status, raw) {
			return nil, ErrChallenge
		}
		return nil, fmt.Errorf("status %d: %s", status, snippet(raw))
	}
	if msg := extractString(stream, "error"); msg != "" {
		return nil, fmt.Errorf("fluz error: %s", msg)
	}
	return stream, nil
}

func (c *Client) tryRelogin() error {
	if c.relogin == nil {
		return ErrChallenge
	}
	cookie, err := c.relogin()
	if err != nil {
		return fmt.Errorf("relogin: %w", err)
	}
	c.sess.merge(parseCookieHeader(cookie))
	return c.sess.Save()
}

func (c *Client) doRequest(method, path string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.endpoint+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("cookie", c.sess.Header())
	req.Header.Set("accept", "*/*")
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("origin", c.endpoint)
	req.Header.Set("referer", c.endpoint+"/cards/new-cashback-card")
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	if c.sess.applySetCookies(resp.Cookies()) {
		_ = c.sess.Save()
	}
	return resp.StatusCode, raw, nil
}

func isChallenge(status int, raw []byte) bool {
	if status == 401 || status == 403 || status == 429 || status == 503 {
		return true
	}
	s := strings.ToLower(string(raw))
	return strings.Contains(s, "just a moment") || strings.Contains(s, "cf-challenge") || strings.Contains(s, "/cdn-cgi/challenge")
}

func snippet(b []byte) string {
	if len(b) > 200 {
		return string(b[:200])
	}
	return string(b)
}

func extract(stream []any, key string) (any, bool) {
	for i := 0; i < len(stream)-1; i++ {
		if s, ok := stream[i].(string); ok && s == key {
			return stream[i+1], true
		}
	}
	return nil, false
}

func extractString(stream []any, key string) string {
	if v, ok := extract(stream, key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractInt(stream []any, key string) int {
	if v, ok := extract(stream, key); ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return 0
}
