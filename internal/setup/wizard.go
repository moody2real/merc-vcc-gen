package setup

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/term"

	"github.com/moody2real/merc-vcc-gen/internal/config"
)

func Needed(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func Run(path string) (*config.Config, error) {
	in := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("  merc-vcc-gen setup")
	fmt.Println("  ------------------")
	fmt.Println("  No config found. Let's create one. Takes about 2 minutes.")
	fmt.Println("  Your answers are saved locally to config.json (only you can read it).")
	fmt.Println()

	var cfg config.Config

	provider := choice(in, "  Which provider?\n    [1] Mercury\n    [2] Fluz\n  > ", map[string]string{
		"1": "mercury", "2": "fluz",
	})
	cfg.Provider = provider

	switch provider {
	case "mercury":
		cfg.Mercury.Email = ask(in, "  Mercury email: ")
		cfg.Mercury.Password = askSecret("  Mercury password: ")
		cfg.Mercury.TOTPSecret = askSecret("  Mercury 2FA (TOTP) secret [see docs/mercury.md]: ")
		cfg.Browser.Headless = yesNo(in, "  Run login browser hidden (headless)? [Y/n]: ", true)
	case "fluz":
		cfg.Fluz.APIKey = askSecret("  Fluz API key: ")
		cfg.Fluz.UserID = ask(in, "  Fluz user ID: ")
		cfg.Fluz.AccountID = ask(in, "  Fluz account ID: ")
		cfg.Fluz.SeatID = ask(in, "  Fluz seat ID (optional, press Enter to skip): ")
	}

	if yesNo(in, "  Use a proxy? [y/N]: ", false) {
		cfg.Proxy.Host = ask(in, "    Proxy host: ")
		cfg.Proxy.Port = askInt(in, "    Proxy port: ", 0)
		cfg.Proxy.Username = ask(in, "    Proxy username (optional): ")
		cfg.Proxy.Password = askSecret("    Proxy password (optional): ")
	}

	cfg.Card.Zip = askDefault(in, "  Billing ZIP", "10001")
	cfg.Card.DailyLimit = askInt(in, "  Daily card limit in dollars [1000]: ", 1000)
	cfg.Card.Nickname = askDefault(in, "  Card nickname", "card")
	cfg.Card.DelaySeconds = 1

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := config.Save(path, &cfg); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	fmt.Println()
	fmt.Printf("  Saved to %s (locked to your user only).\n", path)
	fmt.Println()
	return &cfg, nil
}

func ask(in *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := in.ReadString('\n')
	return strings.TrimSpace(line)
}

func askDefault(in *bufio.Reader, label, def string) string {
	v := ask(in, fmt.Sprintf("%s [%s]: ", label, def))
	if v == "" {
		return def
	}
	return v
}

func askInt(in *bufio.Reader, prompt string, def int) int {
	v := ask(in, prompt)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		fmt.Println("    not a number, using", def)
		return def
	}
	return n
}

func askSecret(prompt string) string {
	fmt.Print(prompt)
	b, err := term.ReadPassword(uintptr(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func yesNo(in *bufio.Reader, prompt string, def bool) bool {
	v := strings.ToLower(ask(in, prompt))
	if v == "" {
		return def
	}
	return v == "y" || v == "yes"
}

func choice(in *bufio.Reader, prompt string, valid map[string]string) string {
	for {
		v := ask(in, prompt)
		if mapped, ok := valid[v]; ok {
			return mapped
		}
		fmt.Println("  invalid choice, try again")
	}
}
