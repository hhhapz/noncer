package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall"

	"github.com/BrianLeishman/go-imap"

	"os"
	"os/signal"

	"github.com/bwmarrin/lit"
	"github.com/hhhapz/noncer/announcements"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}

var (
	verbose   = flag.Bool("v", false, "verbose imap output")
	user      = flag.String("user", "", "imap username")
	pass      = flag.String("pass", "", "imap password")
	host      = flag.String("host", "imap.transip.email", "imap host")
	port      = flag.Int("port", 993, "imap port")
	period    = flag.Int("period", 60, "email fetch period")
	webhook   = flag.String("webhook", "", "webhook url")
	allowList allowListFlags
)

type allowListFlags []string

func (i *allowListFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *allowListFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func run(ctx context.Context) error {
	flag.Var(&allowList, "allowlist", "allow list for source email addresses")
	flag.Parse()

	announcements.AllowedDomains = allowList

	if *webhook == "" {
		return fmt.Errorf("webhook url must be provided")
	}

	// Always Log info level entries
	lit.LogLevel = 4

	imap.Verbose = *verbose
	imap.RetryCount = 3
	im, err := imap.New(*user, *pass, *host, *port)
	if err != nil {
		return fmt.Errorf("could not connect to imap: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	anns := make(chan announcements.Announcement)
	// push announcements to webhook url
	go func() {
		for a := range anns {
			if err := sendWebhook(ctx, a); err != nil {
				lit.Error("sending webhook: %v", err)
			}
		}
	}()
	// listen for announcements
	go func() {
		announcements.Listen(ctx, im, anns, *period)
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	cancel()
	return nil
}

func sendWebhook(ctx context.Context, a announcements.Announcement) error {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(Webhook{fmt.Sprintf("**%s**\n\n%s", a.Subject, a.Contents)})
	req, err := http.NewRequestWithContext(ctx, "POST", *webhook, buf)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf.Reset()
		io.Copy(buf, resp.Body)
		return fmt.Errorf("unexpected status code %d:\n%v", resp.StatusCode, buf.String())
	}
	return nil
}

type Webhook struct {
	Content string `json:"content"`
}
