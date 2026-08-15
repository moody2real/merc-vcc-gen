package fluzcli

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
	"github.com/moody2real/merc-vcc-gen/pkg/fluz"
)

type App struct {
	cfg        *config.Config
	client     *fluz.Client
	in         *bufio.Scanner
	outputPath string
}

func New(cfg *config.Config, client *fluz.Client, outputPath string) *App {
	return &App{
		cfg:        cfg,
		client:     client,
		in:         bufio.NewScanner(os.Stdin),
		outputPath: outputPath,
	}
}

func (a *App) EnsureSession() error {
	return a.client.EnsureToken()
}

func clearScreen() { fmt.Print("\033[H\033[2J\033[3J") }

func (a *App) read() string {
	if !a.in.Scan() {
		return ""
	}
	return strings.TrimSpace(a.in.Text())
}

func (a *App) pause() {
	fmt.Print("\n    Press Enter to return to menu...")
	a.read()
}

func (a *App) renderHeader() {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true).Render("Fluz VCC Gen")
	fmt.Println()
	fmt.Println("    " + title)
	fmt.Println()

	w, err := a.client.Balance()
	if err != nil {
		fmt.Println("    balance unavailable")
		return
	}
	cash := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(w.Balances.CashBalance.AvailableBalance)
	rewards := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(w.Balances.RewardsBalance.AvailableBalance)
	fmt.Printf("    cash %s   rewards %s   bank accounts %d\n", cash, rewards, len(w.BankAccounts))
}

func (a *App) Run() error {
	for {
		clearScreen()
		a.renderHeader()
		fmt.Println()
		fmt.Println("    [1] Create cards")
		fmt.Println("    [2] List funding sources")
		fmt.Println("    [3] Reveal a card by ID")
		fmt.Println("    [4] Exit")
		fmt.Print("\n    > ")

		switch a.read() {
		case "1":
			a.create()
			a.pause()
		case "2":
			a.fundingSources()
			a.pause()
		case "3":
			a.reveal()
			a.pause()
		case "4":
			return nil
		case "":
			return nil
		}
	}
}

func (a *App) fundingSources() {
	banks, err := a.client.FundingSources()
	if err != nil {
		log.Error("could not list funding sources", "err", err)
		return
	}
	log.Info("funding sources", "count", len(banks))
	for _, b := range banks {
		log.Info("bank account", "id", b.BankAccountID, "type", b.Type, "name", b.AccountName, "last4", b.LastFour, "status", b.Status)
	}
}

func (a *App) create() {
	fmt.Print("\n    Merchant slug (e.g. amazon): ")
	slug := a.read()
	if slug == "" {
		log.Error("merchant slug required")
		return
	}

	fmt.Print("    Amount per card (USD): ")
	amount, err := strconv.ParseFloat(a.read(), 64)
	if err != nil || amount <= 0 {
		log.Error("invalid amount")
		return
	}

	fmt.Print("    How many cards? ")
	count, err := strconv.Atoi(a.read())
	if err != nil || count < 1 {
		log.Error("invalid count")
		return
	}

	funding := fluz.FundingFluzBalance
	bankID := ""
	if yes(a.read, "    Fund from a bank account instead of Fluz balance? [y/N]: ") {
		banks, err := a.client.FundingSources()
		if err != nil {
			log.Error("could not list bank accounts", "err", err)
			return
		}
		for _, b := range banks {
			if b.Type == "CHECKING" {
				bankID = b.BankAccountID
				break
			}
		}
		if bankID == "" {
			log.Error("no CHECKING bank account found")
			return
		}
		funding = fluz.FundingBankAccount
	}

	offerID, err := a.client.OfferQuote(slug, amount)
	if err != nil {
		log.Error("offer quote failed", "err", err)
		return
	}

	log.Info("creating cards", "count", count, "merchant", slug, "amount", amount)
	order, err := a.client.CreateAndWait(fluz.CreateRequest{
		OfferID: offerID,
		Items: []fluz.OrderItem{{
			Quantity:             count,
			SpendLimit:           amount,
			SpendLimitDuration:   fluz.DurationLifetime,
			PrimaryFundingSource: funding,
			BankAccountID:        bankID,
			CardNickname:         a.cfg.Card.Nickname,
		}},
	}, 120*time.Second)
	if err != nil {
		log.Error("create failed", "err", err)
		return
	}

	log.Info("order complete", "status", order.OrderStatus, "created", fmt.Sprintf("%d/%d", order.Successful, order.Total))
	for _, card := range order.VirtualCards {
		line := card.Line()
		log.Info("created", "last4", last4(card.CardNumber), "id", card.VirtualCardID)
		if err := a.appendOutput(line); err != nil {
			log.Warn("could not write output file", "path", a.outputPath, "err", err)
		}
	}
}

func (a *App) reveal() {
	fmt.Print("\n    Virtual card ID: ")
	id := a.read()
	if id == "" {
		return
	}
	d, err := a.client.Reveal(id)
	if err != nil {
		log.Error("reveal failed", "err", err)
		return
	}
	log.Info("revealed", "last4", last4(d.CardNumber), "exp", d.ExpiryMMYY, "holder", d.CardHolderName)
	if err := a.appendOutput(d.Line()); err != nil {
		log.Warn("could not write output file", "path", a.outputPath, "err", err)
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

func yes(read func() string, prompt string) bool {
	fmt.Print(prompt)
	v := strings.ToLower(read())
	return v == "y" || v == "yes"
}

func last4(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[len(s)-4:]
}
