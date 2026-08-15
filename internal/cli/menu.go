package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/moody2real/merc-vcc-gen/internal/config"
	"github.com/moody2real/merc-vcc-gen/internal/mercury"
	"github.com/moody2real/merc-vcc-gen/internal/session"
)

var numColors = map[int]string{1: "10", 2: "9", 3: "12", 4: "11"}

func brack(n int, label string) string {
	num := lipgloss.NewStyle().Foreground(lipgloss.Color(numColors[n])).Bold(true).Render(strconv.Itoa(n))
	return "    [" + num + "] " + label
}

type App struct {
	cfg         *config.Config
	client      *mercury.Client
	in          *bufio.Scanner
	sessionPath string
	outputPath  string
}

func New(cfg *config.Config, sessionPath, outputPath string) *App {
	return &App{
		cfg:         cfg,
		in:          bufio.NewScanner(os.Stdin),
		sessionPath: sessionPath,
		outputPath:  outputPath,
	}
}

func (a *App) EnsureSession() error {
	if sess, err := session.Load(a.sessionPath); err == nil {
		client, err := mercury.NewClient(a.cfg, sess)
		if err == nil && client.Verify() == nil {
			a.client = client
			log.Info("reused saved session", "path", a.sessionPath, "savedAt", sess.SavedAt)
			return nil
		}
		log.Warn("saved session invalid or expired, logging in again")
	}

	log.Info("logging in (launching browser through proxy)")
	sess, err := a.login()
	if err != nil {
		return err
	}
	if err := sess.Save(a.sessionPath); err != nil {
		return err
	}
	log.Info("session saved", "path", a.sessionPath)

	client, err := mercury.NewClient(a.cfg, sess)
	if err != nil {
		return err
	}
	if err := client.Verify(); err != nil {
		return fmt.Errorf("session captured but verification failed: %w", err)
	}
	a.client = client
	return nil
}

func (a *App) login() (*session.Session, error) {
	return session.Capture(a.cfg, a.cfg.Browser.Headless)
}

func clearScreen() {
	fmt.Print("\033[H\033[2J\033[3J")
}

func (a *App) fetchBalances() []mercury.AccountBalance {
	balances, err := a.client.Balance()
	if err != nil {
		log.Warn("could not fetch balance", "err", err)
		return nil
	}
	return balances
}

func (a *App) renderHeader(balances []mercury.AccountBalance) {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true).Render("Mercury VCC Gen")
	fmt.Println()
	fmt.Println("    " + title)
	fmt.Println()

	if balances == nil {
		fmt.Println("    balance unavailable")
		return
	}
	for _, acc := range balances {
		name := acc.Name
		if acc.Nickname != "" {
			name = acc.Name + " (" + acc.Nickname + ")"
		}
		avail := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(fmt.Sprintf("$%.2f", acc.AvailableBalance))
		cur := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf("$%.2f", acc.CurrentBalance))
		fmt.Printf("    %-16s available %s   current %s\n", name, avail, cur)
	}
}

func (a *App) pause() {
	fmt.Print("\n    Press Enter to return to menu...")
	a.read()
}

func (a *App) Run() error {
	balances := a.fetchBalances()
	status := ""

	for {
		clearScreen()
		a.renderHeader(balances)
		if status != "" {
			fmt.Println("\n    " + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(status))
		}
		fmt.Println()
		fmt.Println(brack(1, "Create cards"))
		fmt.Println(brack(2, "Delete cards"))
		fmt.Println(brack(3, "List cards"))
		fmt.Println(brack(4, "Exit"))
		fmt.Print("\n    > ")

		switch a.read() {
		case "1":
			a.create()
			balances = a.fetchBalances()
			status = ""
			a.pause()
		case "2":
			a.delete()
			status = ""
			a.pause()
		case "3":
			a.list()
			status = ""
			a.pause()
		case "4":
			return nil
		default:
			status = "invalid option"
		}
	}
}

func (a *App) delay() {
	d := time.Duration(a.cfg.Card.DelaySeconds * float64(time.Second))
	if d <= 0 {
		return
	}
	log.Debug("waiting between requests", "delay", d)
	time.Sleep(d)
}

func (a *App) create() {
	fmt.Print("\n    How many cards to create? ")
	n, err := strconv.Atoi(a.read())
	if err != nil || n < 1 {
		log.Error("invalid number")
		return
	}

	log.Info("creating cards", "count", n, "delaySeconds", a.cfg.Card.DelaySeconds)
	for i := 1; i <= n; i++ {
		if i > 1 {
			a.delay()
		}

		log.Debug("issuing card", "index", i, "of", n)
		card, err := a.client.Issue()
		if err != nil {
			log.Error("issue failed", "index", i, "of", n, "err", err)
			continue
		}
		log.Debug("issued", "index", i, "of", n, "id", card.ID, "last4", card.Last4Digits)

		details, err := a.client.Reveal(card.ID)
		if err != nil {
			log.Error("reveal failed", "index", i, "id", card.ID, "last4", card.Last4Digits, "err", err)
			continue
		}

		line := details.Line()
		log.Info("created", "index", fmt.Sprintf("%d/%d", i, n), "last4", card.Last4Digits)
		if err := a.appendOutput(line); err != nil {
			log.Warn("could not write output file", "path", a.outputPath, "err", err)
		}
	}
	log.Info("create complete", "requested", n)
}

func (a *App) delete() {
	fmt.Println()
	fmt.Println(brack(1, "Delete by last 4 digits"))
	fmt.Println(brack(2, "Delete ALL active cards"))
	fmt.Print("\n    > ")
	mode := a.read()

	cards, err := a.client.List()
	if err != nil {
		log.Error("list failed", "err", err)
		return
	}

	var targets []mercury.Card
	switch mode {
	case "1":
		fmt.Print("    Enter last 4 digits (comma separated): ")
		wanted := splitCSV(a.read())
		for _, c := range cards {
			if c.FullStatus.Tag == "active" && contains(wanted, c.Last4Digits) {
				targets = append(targets, c)
			}
		}
	case "2":
		for _, c := range cards {
			if c.FullStatus.Tag == "active" {
				targets = append(targets, c)
			}
		}
	default:
		log.Warn("invalid option")
		return
	}

	if len(targets) == 0 {
		log.Warn("no matching active cards found")
		return
	}

	fmt.Printf("\n    About to cancel %d card(s). Type 'yes' to confirm: ", len(targets))
	if a.read() != "yes" {
		log.Info("delete cancelled by user")
		return
	}

	log.Info("deleting cards", "count", len(targets), "delaySeconds", a.cfg.Card.DelaySeconds)
	deleted := 0
	for i, c := range targets {
		if i > 0 {
			a.delay()
		}

		if err := a.client.Cancel(c.ID); err != nil {
			log.Error("cancel failed", "last4", c.Last4Digits, "id", c.ID, "err", err)
			continue
		}
		deleted++
		log.Info("cancelled", "last4", c.Last4Digits, "id", c.ID)
	}
	log.Info("delete complete", "deleted", deleted, "total", len(targets))
}

func (a *App) list() {
	cards, err := a.client.List()
	if err != nil {
		log.Error("list failed", "err", err)
		return
	}

	active := 0
	for _, c := range cards {
		if c.FullStatus.Tag == "active" {
			active++
		}
	}

	log.Info("card summary", "total", len(cards), "active", active)
	for _, c := range cards {
		if c.FullStatus.Tag != "active" {
			continue
		}
		log.Info("active card", "last4", c.Last4Digits, "realm", c.CardRealm, "status", c.FullStatus.Tag, "id", c.ID, "nickname", c.Nickname, "created", c.CreatedAtDateStr)
	}
}

func (a *App) appendOutput(line string) error {
	f, err := os.OpenFile(a.outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, line)
	return err
}

func (a *App) read() string {
	if !a.in.Scan() {
		return ""
	}
	return strings.TrimSpace(a.in.Text())
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
