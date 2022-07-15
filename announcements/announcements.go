package announcements

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BrianLeishman/go-imap"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/bwmarrin/lit"
)

const discordMessageSize = 1990

type Announcement struct {
	Subject  string
	Contents []string
}

func Listen(ctx context.Context, im *imap.Dialer, announcements chan<- Announcement, period int) error {
	t := time.NewTicker(time.Duration(period) * time.Second)
	for {
		select {
		case <-ctx.Done():
			close(announcements)
			return ctx.Err()
		case <-t.C:
			err := SelectFolder(im, "INBOX")
			if err != nil {
				return fmt.Errorf("could not select inbox: %v", err)
			}

			// gets all emails in current inbox
			emails, err := im.GetEmails()
			if err != nil {
				lit.Error("get inbox emails: %v", err)
				continue
			}

			var toRemove []int
			for uid, email := range emails {
				toRemove = append(toRemove, uid)
				if !valid(email) {
					lit.Info("received invalid email from %s", email.From)
					continue
				}

				body, err := markdownBody(email.HTML)
				if err != nil {
					lit.Error("create announcement: %v", err)
					continue
				}

				announcements <- Announcement{
					Subject:  email.Subject,
					Contents: buildContents(discordMessageSize, email.Subject, body),
				}
			}

			// no emails were found, so dont bother removing nothing
			if toRemove == nil {
				continue
			}
			if err := remove(im, toRemove); err != nil {
				lit.Error("removing emails: %v", err)
			}
		}
	}
}

// SelectFolder selects a folder
//
// This function replaces the SelectFolder function defined on *imap.Dialer, to
// use the SELECT command instead of the EXTRACT command.
func SelectFolder(im *imap.Dialer, folder string) (err error) {
	_, err = im.Exec(`SELECT "`+imap.AddSlashes.Replace(folder)+`"`, true, imap.RetryCount, nil)
	if err != nil {
		return
	}
	im.Folder = folder
	return nil
}

var AllowedDomains []string

func valid(email *imap.Email) bool {
	for addr := range email.From {
		var ok bool
		for _, domain := range AllowedDomains {
			if strings.HasSuffix(addr, domain) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func remove(im *imap.Dialer, uids []int) error {
	for _, uid := range uids {
		_, err := im.Exec(fmt.Sprintf("UID STORE %d +FLAGS.SILENT (\\Deleted)", uid), false, imap.RetryCount, nil)
		if err != nil {
			return nil
		}
	}
	_, err := im.Exec("EXPUNGE", false, imap.RetryCount, nil)
	return err
}

var converter = md.NewConverter("", true, &md.Options{
	HeadingStyle:    "setext",
	StrongDelimiter: "**",
	LinkStyle:       "inlined",
})

// markdownBody takes a html string and converts it into a formatted markdown body.
//
// markdownBody also removes the signature and replaces double new lines with
// single new lines.
func markdownBody(raw string) (string, error) {
	body, err := converter.ConvertString(raw)
	if err != nil {
		return "", fmt.Errorf("convert to html: %v", err)
	}

	signature := strings.LastIndex(body, "\\-\\-")
	if signature != -1 {
		body = body[:signature]
	}
	body = strings.ReplaceAll(body, "\n\n", "\n")
	return strings.TrimSpace(body), nil
}

// buildContents splits body into chunks so each fit within maxLen.
//
// The size of any element Contents will never exceed maxLen. For the first
// item in Contents, Subject is also considered as part of the contents.
func buildContents(maxLen int, subject, body string) (contents []string) {
	max := maxLen - len(subject)
	for max < len(body) {
		body = strings.TrimSpace(body)

		i := strings.LastIndexAny(body[:max], "\n.!")
		// there are no new lines in the first max len chars
		if i == -1 {
			i = max - 1
		}

		// use i+1 to include the last indexed character
		contents = append(contents, body[:i+1])
		body = body[i+1:]
		max = maxLen
	}
	if body != "" {
		contents = append(contents, body)
	}
	return contents
}
