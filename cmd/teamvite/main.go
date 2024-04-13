package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/benprew/teamvite"
	http "github.com/benprew/teamvite/http"
	"github.com/benprew/teamvite/reminders"
	"github.com/benprew/teamvite/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// Build version, injected during build.
var (
	version string
	commit  string
)

const DefaultConfigPath = "config.json"

// main is the entry point to our application binary. However, it has some poor
// usability so we mainly use it to delegate out to our Main type.
func main() {
	// Propagate build information to root package to share globally.
	teamvite.Version = strings.TrimPrefix(version, "")
	teamvite.Commit = commit

	// Instantiate a new type to represent our application.
	// This type lets us shared setup code with our end-to-end tests.

	var configPath string

	servCmd := flag.NewFlagSet("serv", flag.ExitOnError)
	servCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s serv:\n", servCmd.Name())
		servCmd.PrintDefaults()
		os.Exit(1)
	}
	// Define flags for each subcommand
	// For example, if your "serv" command accepts a "port" flag, you could do:
	servPort := servCmd.String("port", "8080", "port to serve on")
	servCmd.StringVar(&configPath, "config", teamvite.DefaultConfigPath, "config path")

	resetPasswordCmd := flag.NewFlagSet("resetpassword", flag.ExitOnError)
	resetPasswordCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", resetPasswordCmd.Name())
		resetPasswordCmd.PrintDefaults()
		os.Exit(1)
	}
	// If your "resetpassword" command accepts "user" and "newpassword" flags, you could do:
	resetEmail := resetPasswordCmd.String("email", "", "email of user to reset")
	resetNewPassword := resetPasswordCmd.String("newpassword", "", "new password")
	resetPasswordCmd.StringVar(&configPath, "config", teamvite.DefaultConfigPath, "config path")

	sendRemindersCmd := flag.NewFlagSet("sendreminders", flag.ExitOnError)
	sendRemindersCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", sendRemindersCmd.Name())
		sendRemindersCmd.PrintDefaults()
		os.Exit(1)
	}
	sendRemindersCmd.StringVar(&configPath, "config", teamvite.DefaultConfigPath, "config path")

	m := newMain(configPath)
	fmt.Println("config: ", m)

	// Check which subcommand is invoked
	if len(os.Args) < 2 {
		cmdUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serv":
		servCmd.Parse(os.Args[2:])
		// Use the flags, for example:
		fmt.Println("  port:", *servPort)
		m.HTTPServer.Addr = ":" + *servPort
		serv(m)
	case "resetpassword":
		resetPasswordCmd.Parse(os.Args[2:])
		if *resetEmail == "" || *resetNewPassword == "" {
			fmt.Println("Error: email and new password required")
			resetPasswordCmd.Usage()
			os.Exit(1)
		}
		cmdResetPassword(m, *resetEmail, *resetNewPassword)

	case "sendreminders":
		sendRemindersCmd.Parse(os.Args[2:])
		cmdSendReminders(m)
	default:
		cmdUsage()
		os.Exit(1)
	}
}

func cmdResetPassword(m *Main, resetEmail, resetNewPassword string) {
	// Load config and player service
	ps := sqlite.NewPlayerService(m.DB)

	p, len, err := ps.FindPlayers(context.TODO(), teamvite.PlayerFilter{Email: resetEmail, Limit: 1})
	if err != nil {
		fmt.Println("Error finding player", err)
		os.Exit(1)
	}
	if len == 0 {
		fmt.Println("Error: player not found:", resetEmail)
		os.Exit(1)
	}
	err = ps.ResetPassword(teamvite.NewContextWithPlayer(context.TODO(), "", p[0]), resetNewPassword)
	if err != nil {
		fmt.Println("Error resetting password: ", err)
		os.Exit(1)
	}
	fmt.Println("Password reset for", resetEmail)

}

func cmdSendReminders(m *Main) {
	s := reminders.NewReminderService(m.DB, m.Config.SMTP, m.Config.SMS, m.Config.Servername)
	err := s.SendGameReminders()
	if err != nil {
		log.Fatal("Error sending reminders: ", err)
	}
}

func cmdUsage() {
	fmt.Print(`
teamvite - control teamvite server

commands:
	serv           - start the server
	resetpassword  - reset a user's password
	sendreminders  - send game reminders to teams

global options:
	-[h]elp        - print help and exit
	-config <path> - path to config file
`)
}

func serv(m *Main) {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Execute program.
	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		teamvite.ReportError(ctx, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// Clean up program.
	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the program.
type Main struct {
	DB *sql.DB

	// Configuration path and parsed config data.
	Config teamvite.Config

	// SQLite database used by SQLite service implementations.
	// DB *sqlite.DB

	// HTTP server for handling HTTP communication.
	// SQLite services are attached to it before running.
	HTTPServer *http.Server

	// Services exposed for end-to-end tests.
	// UserService wtf.UserService
}

// newMain returns a new instance of Main.
func newMain(configPath string) *Main {
	config, err := teamvite.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	return &Main{
		Config:     config,
		HTTPServer: http.NewServer(),
		DB:         sqlite.Open("file:teamvite.db?_foreign_keys=1"),
	}
}

// func (m *Main) ParseFlags(ctx context.Context, args []string) error {
// 	fs := flag.NewFlagSet("teamvited", flag.ContinueOnError)
// 	fs.StringVar(&m.ConfigPath, "config", DefaultConfigPath, "config path")
// 	log.Println("ConfigPath: ", m.ConfigPath)
// 	log.Println("args: ", args)
// 	if err := fs.Parse(args); err != nil {
// 		return err
// 	}

// 	config, err := teamvite.LoadConfig(m.ConfigPath)
// 	if os.IsNotExist(err) {
// 		return fmt.Errorf("config file not found: %s", m.ConfigPath)
// 	} else if err != nil {
// 		return err
// 	}
// 	m.Config = config
// 	teamvite.CONFIG = config

// 	return nil
// }

func (m *Main) Run(ctx context.Context) error {
	db := sqlite.Open("file:teamvite.db?_foreign_keys=1")

	m.HTTPServer.GameService = sqlite.NewGameService(db)
	m.HTTPServer.TeamService = sqlite.NewTeamService(db)
	m.HTTPServer.PlayerService = sqlite.NewPlayerService(db)
	m.HTTPServer.DivisionService = sqlite.NewDivisionService(db)
	m.HTTPServer.SeasonService = sqlite.NewSeasonService(db)

	m.HTTPServer.SessionService = sqlite.NewSessionService(db)

	// if len(os.Args) == 1 || os.Args[1] == "serv" {
	fmt.Printf("Starting teamvite server on port 8080\n")
	go func() { m.HTTPServer.Open() }()
	// } else if os.Args[1] == "resetpassword" {
	// 	// if err := ResetPassword(db, os.Args[2], os.Args[3]); err != nil {
	// 	// 	fmt.Printf("Error: %v", err)
	// 	// }
	// } else {
	// 	fmt.Printf("ERROR: unknown command %s\n", os.Args[1])
	// }
	return nil
}

func (m *Main) Close() error {
	if m.DB != nil {
		return m.DB.Close()
	}
	return nil
}
