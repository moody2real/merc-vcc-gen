package mercury

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/charmbracelet/log"

	"github.com/moody2real/merc-vcc-gen/internal/config"
	"github.com/moody2real/merc-vcc-gen/internal/session"
)

const backend = "https://backend.mercury.com"

type Client struct {
	cfg     *config.Config
	sess    *session.Session
	http    *http.Client
	account string
	user    string
}

func NewClient(cfg *config.Config, sess *session.Session) (*Client, error) {
	transport := &http.Transport{}
	if cfg.Proxy.Enabled() {
		proxyURL, err := url.Parse(cfg.Proxy.URL())
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &Client{
		cfg:  cfg,
		sess: sess,
		http: &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) do(method, path string, body any) (int, []byte, error) {
	var reader io.Reader
	var bodyBytes []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyBytes = data
		reader = bytes.NewReader(data)
	}

	log.Debug("http request", "method", method, "url", backend+path, "body", string(bodyBytes))

	req, err := http.NewRequest(method, backend+path, reader)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-csrf-protect", "1")
	req.Header.Set("x-xsrf-token", c.sess.XSRF)
	req.Header.Set("cookie", c.sess.Cookie)
	req.Header.Set("origin", "https://app.mercury.com")
	req.Header.Set("user-agent", "Mozilla/5.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Debug("http response", "method", method, "path", path, "status", resp.StatusCode, "bytes", len(data))
	} else {
		log.Debug("http response", "method", method, "path", path, "status", resp.StatusCode, "body", string(data))
	}

	return resp.StatusCode, data, nil
}

func (c *Client) Verify() error {
	status, _, err := c.do(http.MethodPost, "/cards/list", map[string]any{})
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return fmt.Errorf("session invalid (status %d)", status)
	}
	if status != http.StatusOK {
		return fmt.Errorf("unexpected status %d", status)
	}
	return nil
}

func (c *Client) loadIDs() error {
	if c.account != "" && c.user != "" {
		return nil
	}

	status, data, err := c.do(http.MethodGet, "/get-initial-data-v2", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("get-initial-data status %d", status)
	}

	var d initialData
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	if len(d.OrgSpecifics.GlobalData.MercuryDepositoryAccounts) == 0 {
		return fmt.Errorf("no depository account found")
	}

	c.account = d.OrgSpecifics.GlobalData.MercuryDepositoryAccounts[0].ID
	c.user = d.UserDetails.ID
	return nil
}

func (c *Client) Balance() ([]AccountBalance, error) {
	status, data, err := c.do(http.MethodGet, "/get-initial-data-v2", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get-initial-data status %d", status)
	}

	var d initialData
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	accounts := d.OrgSpecifics.GlobalData.MercuryDepositoryAccounts
	out := make([]AccountBalance, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, AccountBalance{
			Name:             a.Name,
			Nickname:         a.Nickname,
			AvailableBalance: a.AvailableBalance,
			CurrentBalance:   a.CurrentBalance,
		})
	}
	return out, nil
}

func (c *Client) List() ([]Card, error) {
	status, data, err := c.do(http.MethodPost, "/cards/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list status %d", status)
	}

	var resp listResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	cards := make([]Card, 0, len(resp.Cards))
	for _, e := range resp.Cards {
		cards = append(cards, e.Contents)
	}
	return cards, nil
}

func (c *Client) Issue() (Card, error) {
	if err := c.loadIDs(); err != nil {
		return Card{}, err
	}

	req := issueRequest{
		CardRealm:      "virtual",
		CardLimits:     cardLimits{Tag: "daily", Contents: cardLimitContents{TransactionLimit: c.cfg.Card.DailyLimit}},
		AccountID:      c.account,
		UserID:         c.user,
		Nickname:       c.cfg.Card.Nickname,
		IssuingAddress: issuingAddress{Tag: "useOrgAddress"},
		CategoryLocks:  []string{},
	}

	status, data, err := c.do(http.MethodPost, "/cards/debit/issue", req)
	if err != nil {
		return Card{}, err
	}
	if status != http.StatusOK {
		return Card{}, fmt.Errorf("issue status %d: %s", status, string(data))
	}

	var resp issueResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return Card{}, err
	}
	return resp.DebitCardData, nil
}

func (c *Client) Cancel(id string) error {
	status, data, err := c.do(http.MethodPost, "/cards/debit/"+id+"/cancel", map[string]any{})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("cancel status %d: %s", status, string(data))
	}
	return nil
}
