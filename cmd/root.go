// Package cmd defines the root Cobra command and CLI flags.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	flagRaw    bool
	flagExport string
	flagOutput string
	version    = "0.1.0"
)

var rootCmd = &cobra.Command{
	Use:     "wwlog",
	Short:   "Browse and export your food log",
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
	rootCmd.Flags().StringVarP(&flagTLD, "tld", "l", "com", "Service top-level domain (com, co.uk, etc.)")
	rootCmd.Flags().BoolVar(&flagRaw, "raw", false, "Dump raw API JSON for the start date (for API inspection)")
	_ = rootCmd.Flags().MarkHidden("raw")
	rootCmd.Flags().StringVar(&flagExport, "export", "", "Export format: json, csv, markdown, report")
	rootCmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output file or directory (default: reports/)")
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

	// Raw API dump for inspection (hidden flag, not shown in --help).
	if flagRaw {
		start := flagStart
		if start == "" {
			start = time.Now().Format("2006-01-02")
		}
		token, err := authenticator.Token()
		if err != nil {
			return fmt.Errorf("%w\nRun 'wwlog --login' to authenticate", err)
		}
		raw, err := api.New(token, tld).FetchDayRaw(start)
		if err != nil {
			return err
		}
		os.Stdout.Write(raw)
		return nil
	}

	// Export mode: write a file without launching the TUI.
	if flagExport != "" {
		extMap := map[string]string{
			"json":     "json",
			"csv":      "csv",
			"markdown": "md",
			"report":   "txt",
		}
		ext, ok := extMap[flagExport]
		if !ok {
			return fmt.Errorf("unknown export format %q — use json, csv, markdown, or report", flagExport)
		}
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
		logs, notices, err := api.LoadRange(api.New(token, tld), start, end)
		for _, n := range notices {
			fmt.Fprintln(os.Stderr, "note:", n)
		}
		if err != nil {
			return err
		}
		dest, err := resolveExportPath(flagOutput, start, end, ext)
		if err != nil {
			return fmt.Errorf("resolve output path: %w", err)
		}
		f, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("create %s: %w", dest, err)
		}
		defer f.Close()
		switch flagExport {
		case "json":
			err = pipeline.WriteJSON(f, logs)
		case "csv":
			err = pipeline.WriteLogCSV(f, logs)
		case "markdown":
			err = pipeline.EmitMarkdown(f, logs)
		case "report":
			err = pipeline.EmitTextReport(f, logs)
		}
		if err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		fmt.Fprintf(os.Stderr, "Saved %s\n", dest)
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
		logs, notices, err := api.LoadRange(client, start, end)
		for _, n := range notices {
			fmt.Fprintln(os.Stderr, "note:", n)
		}
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

func readPassword() (string, error) {
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(b), err
}

// resolveExportPath returns the full destination file path for an export.
// If output names an existing directory (or ends with /) the generated
// filename is appended. If output is empty, reports/ in the cwd is used
// (created on demand). Otherwise output is treated as the literal file path.
func resolveExportPath(output, start, end, ext string) (string, error) {
	filename := fmt.Sprintf("wwlog-%s_%s.%s", start, end, ext)
	if output == "" {
		if err := os.MkdirAll("reports", 0o755); err != nil {
			return "", err
		}
		return filepath.Join("reports", filename), nil
	}
	expanded := output
	if strings.HasPrefix(expanded, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = filepath.Join(home, expanded[2:])
		}
	}
	info, err := os.Stat(expanded)
	if err == nil && info.IsDir() {
		return filepath.Join(expanded, filename), nil
	}
	return expanded, nil
}
