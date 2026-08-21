package tui

import (
	_ "embed"
	"fmt"
	"math/rand/v2"
	"strings"
)

//go:embed assets/exit_messages.txt
var exitMessagesRaw string

// exitMessages are short, upbeat sign-offs shown after the logo when the
// user quits — supportive of someone tracking their food log, not generic
// app boilerplate. Edit assets/exit_messages.txt to change them, one per
// line; blank lines and lines starting with # are ignored.
var exitMessages = parseExitMessages(exitMessagesRaw)

func parseExitMessages(raw string) []string {
	var messages []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		messages = append(messages, line)
	}
	return messages
}

// printExitBanner renders the wwlog logo (logo.go) and a random encouraging
// sign-off to stdout after the TUI has exited. Matches the look prototyped
// in examples/ending.sh.
func printExitBanner() {
	fmt.Println()
	fmt.Println(cfontsLogo)
	fmt.Println()
	msg := exitMessages[rand.IntN(len(exitMessages))]
	fmt.Println(" " + styleExitMessage.Render("Thanks for using wwlog. "+msg))
	fmt.Println(" " + styleExitByline.Render("GitHub · ali5ter/wwlog"))
}
