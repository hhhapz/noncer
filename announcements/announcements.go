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

type Announcement struct {
	Subject  string
	Contents []string
}

var converter = md.NewConverter("", true, &md.Options{
	HeadingStyle:    "setext",
	StrongDelimiter: "**",
	LinkStyle:       "inlined",
})

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

				contents, err := converter.ConvertString(email.HTML)
				if err != nil {
					lit.Error("convert to html: %v", err)
					continue
				}

				announcements <- formatAnnouncement(1990, email.Subject, contents)
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

var replacer = strings.NewReplacer("\n\n", "\n")

// formatAnnouncement formats an announcement from the given subject and contents
//
// it does a slightly ugly hack with maxLen to both make the announcement fit and pretty, so try to keep some leeway.
func formatAnnouncement(maxLen int, subject string, body string) Announcement {
	body = replacer.Replace(body)
	body = strings.TrimSpace(body)

	cutoff := strings.LastIndex(body, "\\-\\-")
	if cutoff != -1 {
		body = body[:cutoff]
		body = strings.TrimSpace(body)
	}

	var contents []string

	currLen := len(subject) + len(body)
	subLen := len(subject)
	newMaxLen := maxLen
	// will this fit into discord?
	for currLen > newMaxLen {
		i := strings.LastIndexAny(body[0:newMaxLen-subLen], "\n.;!?")
		if i == -1 {
			newMaxLen++
			continue
		} else {
			i += 1
		}

		part := strings.TrimSpace(body[:i])
		body = body[i:]
		if part == "" {
			continue
		}

		contents = append(contents, part)

		// subsequent messages don't have the subject
		subLen = 0
		currLen = len(body)
		newMaxLen = maxLen
	}
	if body != "" {
		contents = append(contents, strings.TrimSpace(body))
	}

	return Announcement{Subject: subject, Contents: contents}
}
