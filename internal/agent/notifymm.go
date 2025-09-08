package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zeshi09/go_web_parser_agent/ent"
)

func NotifyMMDomains(webhook string, domains []*ent.Domain) error {
	var b strings.Builder
	b.WriteString("**Появились новые домены:**\n")
	for _, d := range domains {
		b.WriteString("- ")
		b.WriteString(d.LandingDomain)
		b.WriteString("\n")
	}
	payload := map[string]string{
		"text":     b.String(),
		"username": "DomainWatcher",
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("mm webhook returned %s", resp.Status)
	}
	return nil
}

func NotifyMMLinks(webhook string, links []*ent.SocialLink) error {
	var b strings.Builder
	b.WriteString("**Появились новые ссылки:**\n")
	for _, l := range links {
		b.WriteString("- ")
		b.WriteString(l.URL)
		b.WriteString("\n")
	}
	payload := map[string]string{
		"text":     b.String(),
		"username": "LinkWatcher",
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("mm webhook returned %s", resp.Status)
	}
	return nil
}
