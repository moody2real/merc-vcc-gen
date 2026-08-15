package merc

import (
	"fmt"

	"github.com/moody2real/merc-vcc-gen/internal/config"
	"github.com/moody2real/merc-vcc-gen/internal/mercury"
	"github.com/moody2real/merc-vcc-gen/internal/session"
)

type (
	Config         = config.Config
	Mercury        = config.Mercury
	Proxy          = config.Proxy
	CardConfig     = config.Card
	Browser        = config.Browser
	Card           = mercury.Card
	CardDetails    = mercury.CardDetails
	AccountBalance = mercury.AccountBalance
	Session        = session.Session
)

func LoadConfig(path string) (*Config, error) {
	return config.Load(path)
}

type Client struct {
	cfg         *config.Config
	api         *mercury.Client
	sessionPath string
}

func New(cfg *config.Config, sessionPath string) *Client {
	return &Client{cfg: cfg, sessionPath: sessionPath}
}

func (c *Client) EnsureSession() error {
	if sess, err := session.Load(c.sessionPath); err == nil {
		api, err := mercury.NewClient(c.cfg, sess)
		if err == nil && api.Verify() == nil {
			c.api = api
			return nil
		}
	}
	return c.Login()
}

func (c *Client) Login() error {
	sess, err := session.Capture(c.cfg, c.cfg.Browser.Headless)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	if c.sessionPath != "" {
		if err := sess.Save(c.sessionPath); err != nil {
			return fmt.Errorf("save session: %w", err)
		}
	}

	api, err := mercury.NewClient(c.cfg, sess)
	if err != nil {
		return err
	}
	if err := api.Verify(); err != nil {
		return fmt.Errorf("session verification failed: %w", err)
	}
	c.api = api
	return nil
}

func (c *Client) ready() error {
	if c.api == nil {
		return fmt.Errorf("no active session; call EnsureSession or Login first")
	}
	return nil
}

func (c *Client) Balance() ([]AccountBalance, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	return c.api.Balance()
}

func (c *Client) List() ([]Card, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	return c.api.List()
}

func (c *Client) ActiveCards() ([]Card, error) {
	cards, err := c.List()
	if err != nil {
		return nil, err
	}
	active := make([]Card, 0, len(cards))
	for _, card := range cards {
		if card.FullStatus.Tag == "active" {
			active = append(active, card)
		}
	}
	return active, nil
}

func (c *Client) Issue() (Card, error) {
	if err := c.ready(); err != nil {
		return Card{}, err
	}
	return c.api.Issue()
}

func (c *Client) Reveal(id string) (CardDetails, error) {
	if err := c.ready(); err != nil {
		return CardDetails{}, err
	}
	return c.api.Reveal(id)
}

func (c *Client) Create() (Card, CardDetails, error) {
	card, err := c.Issue()
	if err != nil {
		return Card{}, CardDetails{}, err
	}
	details, err := c.Reveal(card.ID)
	if err != nil {
		return card, CardDetails{}, err
	}
	return card, details, nil
}

func (c *Client) Cancel(id string) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.api.Cancel(id)
}

func (c *Client) DeleteAllActive() (int, error) {
	active, err := c.ActiveCards()
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, card := range active {
		if err := c.Cancel(card.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
