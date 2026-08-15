package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Provider string  `json:"provider"`
	Mercury  Mercury `json:"mercury"`
	Fluz     Fluz    `json:"fluz"`
	Proxy    Proxy   `json:"proxy"`
	Card     Card    `json:"card"`
	Browser  Browser `json:"browser"`
	Debug    bool    `json:"debug"`
}

type Mercury struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TOTPSecret string `json:"totpSecret"`
}

type Fluz struct {
	APIKey    string `json:"apiKey"`
	UserID    string `json:"userId"`
	AccountID string `json:"accountId"`
	SeatID    string `json:"seatId"`
}

type Proxy struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Card struct {
	DailyLimit   int     `json:"dailyLimit"`
	Zip          string  `json:"zip"`
	Nickname     string  `json:"nickname"`
	DelaySeconds float64 `json:"delaySeconds"`
}

type Browser struct {
	ChromePath  string `json:"chromePath"`
	UserDataDir string `json:"userDataDir"`
	Headless    bool   `json:"headless"`
}

func (p Proxy) Enabled() bool { return p.Host != "" && p.Port != 0 }

func (p Proxy) URL() string {
	if !p.Enabled() {
		return ""
	}
	u := url.URL{Scheme: "http", Host: net.JoinHostPort(p.Host, strconv.Itoa(p.Port))}
	if p.Username != "" || p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u.String()
}

func (m Mercury) String() string {
	return fmt.Sprintf("Mercury{email:%s password:REDACTED totp:REDACTED}", m.Email)
}

func (f Fluz) String() string {
	return fmt.Sprintf("Fluz{apiKey:REDACTED userId:%s accountId:%s}", f.UserID, f.AccountID)
}

func (p Proxy) String() string {
	if !p.Enabled() {
		return "Proxy{disabled}"
	}
	return fmt.Sprintf("Proxy{host:%s port:%d user:%s password:REDACTED}", p.Host, p.Port, p.Username)
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyEnv()
	return &cfg, nil
}

func (c *Config) applyEnv() {
	if v := os.Getenv("MERC_PROVIDER"); v != "" {
		c.Provider = v
	}
	if v := os.Getenv("MERC_MERCURY_EMAIL"); v != "" {
		c.Mercury.Email = v
	}
	if v := os.Getenv("MERC_MERCURY_PASSWORD"); v != "" {
		c.Mercury.Password = v
	}
	if v := os.Getenv("MERC_MERCURY_TOTP"); v != "" {
		c.Mercury.TOTPSecret = v
	}
	if v := os.Getenv("MERC_FLUZ_API_KEY"); v != "" {
		c.Fluz.APIKey = v
	}
	if v := os.Getenv("MERC_FLUZ_USER_ID"); v != "" {
		c.Fluz.UserID = v
	}
	if v := os.Getenv("MERC_FLUZ_ACCOUNT_ID"); v != "" {
		c.Fluz.AccountID = v
	}
}

func (c *Config) Validate() error {
	var missing []string

	switch strings.ToLower(c.Provider) {
	case "mercury":
		if c.Mercury.Email == "" {
			missing = append(missing, "mercury.email")
		}
		if c.Mercury.Password == "" {
			missing = append(missing, "mercury.password")
		}
		if c.Mercury.TOTPSecret == "" {
			missing = append(missing, "mercury.totpSecret")
		}
	case "fluz":
		if c.Fluz.APIKey == "" {
			missing = append(missing, "fluz.apiKey")
		}
		if c.Fluz.UserID == "" {
			missing = append(missing, "fluz.userId")
		}
		if c.Fluz.AccountID == "" {
			missing = append(missing, "fluz.accountId")
		}
	case "":
		return fmt.Errorf("provider is empty; set it to \"mercury\" or \"fluz\"")
	default:
		return fmt.Errorf("unknown provider %q; use \"mercury\" or \"fluz\"", c.Provider)
	}

	if c.Proxy.Host != "" && c.Proxy.Port == 0 {
		missing = append(missing, "proxy.port")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required config fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

func Save(path string, c *Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
