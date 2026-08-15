package mercury

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/charmbracelet/log"
)

var (
	tagRE   = regexp.MustCompile(`<[^>]*>`)
	digitRE = regexp.MustCompile(`\D`)
	panRE   = regexp.MustCompile(`(?s)<div id="pan"[^>]*>(.*?)</div>`)
	cvvRE   = regexp.MustCompile(`(?s)<div id="cvv"[^>]*>(.*?)</div>`)
	monthRE = regexp.MustCompile(`id="month"[^>]*>(\d{2})`)
	yearRE  = regexp.MustCompile(`id="year"[^>]*>(\d{2})`)
)

func (c *Client) Reveal(id string) (CardDetails, error) {
	var presigned string
	for attempt := 1; attempt <= 8; attempt++ {
		status, data, err := c.do(http.MethodPost, "/cards/debit/"+id+"/embed-reveal", map[string]any{})
		if err != nil {
			return CardDetails{}, err
		}
		if status == http.StatusOK {
			var resp revealResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				return CardDetails{}, err
			}
			presigned = resp.PresignedURL
			log.Debug("reveal ready", "id", id, "attempt", attempt, "presignedURL", presigned)
			break
		}
		log.Debug("card not ready to reveal, retrying", "id", id, "attempt", attempt, "status", status)
		time.Sleep(6 * time.Second)
	}
	if presigned == "" {
		return CardDetails{}, fmt.Errorf("card never became ready to reveal")
	}

	html, err := c.fetchLithic(presigned)
	if err != nil {
		return CardDetails{}, err
	}
	log.Debug("fetched reveal page", "bytes", len(html))

	details, err := parseCardHTML(html, c.cfg.Card.Zip)
	if err != nil {
		return CardDetails{}, err
	}
	log.Debug("parsed card details", "pan", details.PAN, "exp", details.Expiry, "cvv", details.CVV, "zip", details.Zip)
	return details, nil
}

func (c *Client) fetchLithic(presignedURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, presignedURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("referer", "https://app.mercury.com")
	req.Header.Set("user-agent", "Mozilla/5.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lithic status %d", resp.StatusCode)
	}
	return string(data), nil
}

func parseCardHTML(html, zip string) (CardDetails, error) {
	panMatch := panRE.FindStringSubmatch(html)
	cvvMatch := cvvRE.FindStringSubmatch(html)
	monthMatch := monthRE.FindStringSubmatch(html)
	yearMatch := yearRE.FindStringSubmatch(html)

	if panMatch == nil || monthMatch == nil || yearMatch == nil || cvvMatch == nil {
		return CardDetails{}, fmt.Errorf("could not parse card details from reveal page")
	}

	pan := digitRE.ReplaceAllString(tagRE.ReplaceAllString(panMatch[1], ""), "")
	cvv := digitRE.ReplaceAllString(tagRE.ReplaceAllString(cvvMatch[1], ""), "")

	return CardDetails{
		PAN:    pan,
		Expiry: monthMatch[1] + "/" + yearMatch[1],
		CVV:    cvv,
		Zip:    zip,
	}, nil
}
