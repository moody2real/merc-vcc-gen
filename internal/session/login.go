package session

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	"github.com/pquerna/otp/totp"

	"github.com/moody2real/merc-vcc-gen/internal/browser"
	"github.com/moody2real/merc-vcc-gen/internal/config"
)

var lastTOTP string

func logf(format string, args ...any) {
	log.Infof(format, args...)
}

func freshTOTP(secret string) (string, error) {
	for {
		remaining := 30 - time.Now().Unix()%30
		code, err := totp.GenerateCode(secret, time.Now())
		if err != nil {
			return "", err
		}
		if code != lastTOTP && remaining >= 8 {
			lastTOTP = code
			return code, nil
		}
		logf("waiting %ds for a fresh, unused TOTP code...", remaining+1)
		time.Sleep(time.Duration(remaining+1) * time.Second)
	}
}

func Capture(cfg *config.Config, headless bool) (*Session, error) {
	browser.KillStale(cfg.Browser.UserDataDir)

	proxyURL := fmt.Sprintf("http://%s:%d", cfg.Proxy.Host, cfg.Proxy.Port)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer(proxyURL),
		chromedp.UserDataDir(cfg.Browser.UserDataDir),
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("no-sandbox", true),
		chromedp.WindowSize(1920, 1080),
	)
	chromePath, err := browser.Resolve(cfg.Browser.ChromePath)
	if err != nil {
		return nil, fmt.Errorf("resolve chrome: %w", err)
	}
	opts = append(opts, chromedp.ExecPath(chromePath))

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelTimeout()

	logf("launching browser (headless=%v) through proxy %s", headless, proxyURL)
	handleProxyAuth(ctx, cfg)

	if err := chromedp.Run(ctx, fetch.Enable().WithHandleAuthRequests(true)); err != nil {
		return nil, fmt.Errorf("enable fetch: %w", err)
	}

	if err := doLogin(ctx, cfg); err != nil {
		return nil, err
	}

	logf("login reached dashboard, extracting cookies...")
	return extractSession(ctx)
}

func handleProxyAuth(ctx context.Context, cfg *config.Config) {
	chromedp.ListenTarget(ctx, func(ev any) {
		switch e := ev.(type) {
		case *fetch.EventAuthRequired:
			go func() {
				_ = chromedp.Run(ctx, fetch.ContinueWithAuth(e.RequestID, &fetch.AuthChallengeResponse{
					Response: fetch.AuthChallengeResponseResponseProvideCredentials,
					Username: cfg.Proxy.Username,
					Password: cfg.Proxy.Password,
				}))
			}()
		case *fetch.EventRequestPaused:
			go func() {
				_ = chromedp.Run(ctx, fetch.ContinueRequest(e.RequestID))
			}()
		}
	})
}

func doLogin(ctx context.Context, cfg *config.Config) error {
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://app.mercury.com/login"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return fmt.Errorf("navigate login: %w", err)
	}

	var path string
	if err := chromedp.Run(ctx, chromedp.Evaluate(`window.location.pathname`, &path)); err != nil {
		return err
	}
	if !strings.HasPrefix(path, "/login") {
		logf("already logged in (landed on %s), skipping credentials", path)
		return nil
	}

	logf("entering credentials...")
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`#email-form-field`, chromedp.ByQuery),
		chromedp.SendKeys(`#email-form-field`, cfg.Mercury.Email, chromedp.ByQuery),
		chromedp.SendKeys(`#password-form-field`, cfg.Mercury.Password, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	); err != nil {
		return fmt.Errorf("fill credentials: %w", err)
	}

	logf("clicking Log in...")
	if err := clickByText(ctx, "Log in"); err != nil {
		return fmt.Errorf("click login: %w", err)
	}

	if err := handle2FA(ctx, cfg); err != nil {
		return err
	}

	logf("waiting for dashboard...")
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if err := chromedp.Run(ctx, chromedp.Evaluate(`window.location.pathname`, &path)); err != nil {
			return err
		}
		if !strings.HasPrefix(path, "/login") {
			return nil
		}
		time.Sleep(time.Second)
	}

	var bodyText string
	_ = chromedp.Run(ctx, chromedp.Evaluate(`document.body.innerText`, &bodyText))
	return fmt.Errorf("login stuck on %s; page text: %q", path, bodyText)
}

const totpSelector = `input[inputmode="numeric"], input[autocomplete="one-time-code"]`

func handle2FA(ctx context.Context, cfg *config.Config) error {
	hasInput := false
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`!!document.querySelector(`+jsString(totpSelector)+`)`, &hasInput))
		if hasInput {
			break
		}
		time.Sleep(time.Second)
	}
	if !hasInput {
		logf("no 2FA prompt detected")
		return nil
	}

	code, err := freshTOTP(cfg.Mercury.TOTPSecret)
	if err != nil {
		return fmt.Errorf("generate totp: %w", err)
	}
	logf("entering TOTP code %s", code)

	if err := chromedp.Run(ctx,
		chromedp.SendKeys(totpSelector, code, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	); err != nil {
		return fmt.Errorf("enter totp: %w", err)
	}

	return clickByText(ctx, "Log in")
}

func jsString(s string) string {
	return "`" + s + "`"
}

func clickByText(ctx context.Context, text string) error {
	js := fmt.Sprintf(`(() => {
		const btn = [...document.querySelectorAll('button')].find(b => b.textContent.trim().includes(%q) && b.offsetParent !== null);
		if (btn) { btn.click(); return true; }
		return false;
	})()`, text)

	var clicked bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &clicked)); err != nil {
		return err
	}
	if !clicked {
		return fmt.Errorf("button %q not found", text)
	}
	return nil
}

func extractSession(ctx context.Context) (*Session, error) {
	logf("visiting /cards to ensure XSRF cookie is set...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://app.mercury.com/cards"),
		chromedp.Sleep(4*time.Second),
	); err != nil {
		return nil, fmt.Errorf("navigate cards: %w", err)
	}

	var cookies []*network.Cookie
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = storage.GetCookies().Do(ctx)
		return err
	})); err != nil {
		return nil, fmt.Errorf("get cookies: %w", err)
	}

	seen := make(map[string]bool)
	var parts []string
	var xsrf string
	var names []string
	for _, c := range cookies {
		if !strings.Contains(c.Domain, "mercury.com") {
			continue
		}
		if seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		names = append(names, c.Name)
		parts = append(parts, c.Name+"="+c.Value)
		if c.Name == "XSRF-TOKEN" {
			if decoded, err := url.QueryUnescape(c.Value); err == nil {
				xsrf = decoded
			} else {
				xsrf = c.Value
			}
		}
	}
	logf("mercury cookies: %s", strings.Join(names, ", "))

	logf("extracted %d cookies, xsrf=%v", len(parts), xsrf != "")
	if xsrf == "" {
		return nil, fmt.Errorf("XSRF-TOKEN cookie not found after login")
	}

	return &Session{
		Cookie:  strings.Join(parts, "; "),
		XSRF:    xsrf,
		SavedAt: time.Now(),
	}, nil
}
