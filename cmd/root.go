// Package cmd defines the root Cobra command and CLI flags.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ali5ter/wwlog/config"
	"github.com/ali5ter/wwlog/internal/api"
	"github.com/ali5ter/wwlog/internal/auth"
	"github.com/ali5ter/wwlog/internal/pipeline"
	"github.com/ali5ter/wwlog/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	flagStart  string
	flagEnd    string
	flagJSON   bool
	flagReport bool
	flagNoTTY  bool
	flagLogin  bool
	flagLogout bool
	flagTLD    string
	version    = "0.1.0"
)

var rootCmd = &cobra.Command{
	Use:     "wwlog",
	Short:   "Browse and export your Weight Watchers food log",
	Version: version,
	RunE:    run,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&flagStart, "start", "s", "", "Start date (YYYY-MM-DD, default: last 7 days)")
	rootCmd.Flags().StringVarP(&flagEnd, "end", "e", "", "End date (YYYY-MM-DD, default: today)")
	rootCmd.Flags().BoolVar(&flagJSON, "json", false, "Emit log as JSON to stdout")
	rootCmd.Flags().BoolVarP(&flagReport, "report", "r", false, "Emit insights report to stdout")
	rootCmd.Flags().BoolVar(&flagNoTTY, "no-tty", false, "Force pipeline mode even in a terminal")
	rootCmd.Flags().BoolVar(&flagLogin, "login", false, "Authenticate and store credentials")
	rootCmd.Flags().BoolVar(&flagLogout, "logout", false, "Clear stored credentials")
	rootCmd.Flags().StringVarP(&flagTLD, "tld", "l", "com", "WW top-level domain (com, co.uk, etc.)")
}

func run(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v\n", err)
	}

	tld := flagTLD
	if cfg.TLD != "" {
		tld = cfg.TLD
	}

	authenticator := &auth.Auth{TLD: tld}

	if flagLogout {
		if err := authenticator.Logout(); err != nil {
			return fmt.Errorf("logout: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Credentials cleared.")
		return nil
	}

	if flagLogin {
		fmt.Fprint(os.Stderr, "Email: ")
		var email string
		fmt.Scanln(&email)
		fmt.Fprint(os.Stderr, "Password: ")
		password, err := readPassword()
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		token, err := authenticator.Login(email, password)
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}
		msg := "Authenticated and credentials stored."
		if exp, err := authenticator.Expiry(token); err == nil {
			msg += fmt.Sprintf(" Session expires %s.", exp.Local().Format("Mon 2 Jan 2006 at 15:04"))
		}
		fmt.Fprintln(os.Stderr, msg)
		return nil
	}

	// Pipeline mode: requires explicit dates; defaults to today if omitted.
	if flagJSON || flagReport || flagNoTTY || !isTTY() {
		start := flagStart
		if start == "" {
			start = time.Now().Format("2006-01-02")
		}
		end := flagEnd
		if end == "" {
			end = time.Now().Format("2006-01-02")
		}
		token, err := authenticator.Token()
		if err != nil {
			return fmt.Errorf("%w\nRun 'wwlog --login' to authenticate", err)
		}
		client := api.New(token, tld)
		logs, err := loadLogs(client, start, end)
		if err != nil {
			return err
		}
		if flagReport {
			return pipeline.EmitTextReport(os.Stdout, logs)
		}
		return pipeline.EmitJSON(logs)
	}

	// TUI mode: auth and date range handled inside the TUI.
	return tui.Run(authenticator, tld, flagStart, flagEnd, version)
}

func loadLogs(client *api.Client, start, end string) ([]*api.DayLog, error) {
	dates, err := api.DateRange(start, end)
	if err != nil {
		return nil, err
	}
	var logs []*api.DayLog
	for _, date := range dates {
		day, err := client.FetchDay(date)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", date, err)
		}
		logs = append(logs, day)
	}
	return logs, nil
}

func readPassword() (string, error) {
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(b), err
}
