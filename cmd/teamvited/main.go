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
	"github.com/benprew/teamvite/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// Build version, injected during build.
var (
	version string
	commit  string
)

const DefaultConfigPath = "config.json"

// TODO: This should probably be teamvite (without the d) and then one of the
// subcommands can be "daemon" or "server" or "serv" to run the server.
// TODO: add a reset password command
// TODO: add a send game reminders command

// main is the entry point to our application binary. However, it has some poor
// usability so we mainly use it to delegate out to our Main type.
func main() {
	// Propagate build information to root package to share globally.
	teamvite.Version = strings.TrimPrefix(version, "")
	teamvite.Commit = commit

	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Instantiate a new type to represent our application.
	// This type lets us shared setup code with our end-to-end tests.
	m := NewMain()

	// Parse command line flags & load configuration.
	if err := m.ParseFlags(ctx, os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

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
	Config     teamvite.Config
	ConfigPath string

	// SQLite database used by SQLite service implementations.
	// DB *sqlite.DB

	// HTTP server for handling HTTP communication.
	// SQLite services are attached to it before running.
	HTTPServer *http.Server

	// Services exposed for end-to-end tests.
	// UserService wtf.UserService
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		ConfigPath: teamvite.DefaultConfigPath,
		HTTPServer: http.NewServer(),
	}
}

func (m *Main) ParseFlags(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("teamvited", flag.ContinueOnError)
	fs.StringVar(&m.ConfigPath, "config", DefaultConfigPath, "config path")
	log.Println("ConfigPath: ", m.ConfigPath)
	log.Println("args: ", args)
	if err := fs.Parse(args); err != nil {
		return err
	}

	config, err := teamvite.LoadConfig(m.ConfigPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", m.ConfigPath)
	} else if err != nil {
		return err
	}
	m.Config = config
	teamvite.CONFIG = config

	return nil
}

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
