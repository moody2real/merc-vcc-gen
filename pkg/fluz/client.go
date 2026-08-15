package fluz

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultEndpoint = "https://transactional-graph.fluzapp.com/api/v1/graphql"

var DefaultScopes = []string{
	"LIST_PAYMENT",
	"LIST_PURCHASES",
	"LIST_OFFERS",
	"CREATE_VIRTUALCARD",
	"REVEAL_VIRTUALCARD",
	"MANAGE_PAYMENT",
	"EDIT_VIRTUALCARD",
}

type Config struct {
	APIKey    string
	UserID    string
	AccountID string
	SeatID    string
	Endpoint  string
	Scopes    []string
}

type Client struct {
	cfg      Config
	http     *http.Client
	endpoint string
	scopes   []string
	token    string
}

func New(cfg Config) *Client {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}
	return &Client{
		cfg:      cfg,
		http:     &http.Client{Timeout: 30 * time.Second},
		endpoint: endpoint,
		scopes:   scopes,
	}
}

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("fluz api (%d): %s", e.Status, e.Message)
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Error string `json:"error"`
}

func (c *Client) EnsureToken() error {
	if c.token != "" {
		return nil
	}
	return c.generateToken()
}

func (c *Client) generateToken() error {
	const q = `mutation($userId: UUID!, $accountId: UUID!, $scopes: [ScopeType!]!, $seatId: UUID) { generateUserAccessToken(userId: $userId, accountId: $accountId, scopes: $scopes, seatId: $seatId) { token refreshToken scopes } }`
	vars := map[string]any{
		"userId":    c.cfg.UserID,
		"accountId": c.cfg.AccountID,
		"scopes":    c.scopes,
	}
	if c.cfg.SeatID != "" {
		vars["seatId"] = c.cfg.SeatID
	}

	var out struct {
		GenerateUserAccessToken struct {
			Token        string   `json:"token"`
			RefreshToken string   `json:"refreshToken"`
			Scopes       []string `json:"scopes"`
		} `json:"generateUserAccessToken"`
	}
	if err := c.request("Basic "+c.cfg.APIKey, q, vars, &out); err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	if out.GenerateUserAccessToken.Token == "" {
		return fmt.Errorf("generate token: empty token")
	}
	c.token = out.GenerateUserAccessToken.Token
	return nil
}

func (c *Client) gql(query string, vars map[string]any, out any) error {
	if err := c.EnsureToken(); err != nil {
		return err
	}
	err := c.request("Bearer "+c.token, query, vars, out)
	if err != nil && isAuthErr(err) {
		c.token = ""
		if err := c.generateToken(); err != nil {
			return err
		}
		return c.request("Bearer "+c.token, query, vars, out)
	}
	return err
}

func (c *Client) request(auth, query string, vars map[string]any, out any) error {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", auth)
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var gr gqlResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return &APIError{Status: resp.StatusCode, Message: string(body)}
	}
	if len(gr.Errors) > 0 {
		return &APIError{Status: resp.StatusCode, Message: gr.Errors[0].Message}
	}
	if gr.Error != "" {
		return &APIError{Status: resp.StatusCode, Message: gr.Error}
	}
	if len(gr.Data) == 0 {
		return &APIError{Status: resp.StatusCode, Message: "empty response: " + string(body)}
	}
	if out != nil {
		return json.Unmarshal(gr.Data, out)
	}
	return nil
}

func isAuthErr(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		if ae.Status == http.StatusUnauthorized {
			return true
		}
		msg := strings.ToLower(ae.Message)
		return strings.Contains(msg, "invalid access") || strings.Contains(msg, "token")
	}
	return false
}
