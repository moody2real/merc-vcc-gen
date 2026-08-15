package fluzweb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"

	"github.com/moody2real/merc-vcc-gen/internal/browser"
)

type LoginConfig struct {
	ChromePath  string
	UserDataDir string
	ProxyServer string
	ProxyUser   string
	ProxyPass   string
	Timeout     time.Duration
}

func Capture(cfg LoginConfig) (string, error) {
	chromePath, err := browser.Resolve(cfg.ChromePath)
	if err != nil {
		return "", fmt.Errorf("resolve chrome: %w", err)
	}
	browser.KillStale(cfg.UserDataDir)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
		chromedp.UserDataDir(cfg.UserDataDir),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.WindowSize(1280, 900),
	)
	if cfg.ProxyServer != "" {
		opts = append(opts, chromedp.ProxyServer(cfg.ProxyServer))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()

	if cfg.ProxyUser != "" {
		handleProxyAuth(ctx, cfg)
		if err := chromedp.Run(ctx, fetch.Enable().WithHandleAuthRequests(true)); err != nil {
			return "", fmt.Errorf("enable fetch: %w", err)
		}
	}

	if err := chromedp.Run(ctx, chromedp.Navigate("https://fluz.app/")); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var cookies []*network.Cookie
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = storage.GetCookies().Do(ctx)
			return err
		})); err != nil {
			return "", fmt.Errorf("read cookies: %w", err)
		}
		if hasSession(cookies) {
			return buildCookieHeader(cookies), nil
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("login not completed before timeout (no __session cookie)")
}

func handleProxyAuth(ctx context.Context, cfg LoginConfig) {
	chromedp.ListenTarget(ctx, func(ev any) {
		if e, ok := ev.(*fetch.EventAuthRequired); ok {
			go func() {
				_ = chromedp.Run(ctx, fetch.ContinueWithAuth(e.RequestID, &fetch.AuthChallengeResponse{
					Response: fetch.AuthChallengeResponseResponseProvideCredentials,
					Username: cfg.ProxyUser,
					Password: cfg.ProxyPass,
				}))
			}()
		} else if e, ok := ev.(*fetch.EventRequestPaused); ok {
			go func() { _ = chromedp.Run(ctx, fetch.ContinueRequest(e.RequestID)) }()
		}
	})
}

func hasSession(cookies []*network.Cookie) bool {
	for _, c := range cookies {
		if c.Name == "__session" && c.Value != "" {
			return true
		}
	}
	return false
}

func buildCookieHeader(cookies []*network.Cookie) string {
	var parts []string
	for _, c := range cookies {
		if strings.Contains(c.Domain, "fluz.app") {
			parts = append(parts, c.Name+"="+c.Value)
		}
	}
	return strings.Join(parts, "; ")
}
