package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
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
		Config:     teamvite.DefaultConfig(),
		ConfigPath: teamvite.DefaultConfigPath,

		HTTPServer: http.NewServer(),
	}
}

func (m *Main) ParseFlags(ctx context.Context, args []string) error {
	flag.StringVar(&m.ConfigPath, "config", m.ConfigPath, "path to config file")
	flag.Parse()
	return nil
}

func (m *Main) Run(ctx context.Context) error {
	db := sqlite.Open("file:teamvite.db?_foreign_keys=1")
	defer db.Close()

	m.HTTPServer.GameService = sqlite.NewGameService(db)
	m.HTTPServer.TeamService = sqlite.NewTeamService(db)
	m.HTTPServer.PlayerService = sqlite.NewPlayerService(db)
	m.HTTPServer.DivisionService = sqlite.NewDivisionService(db)

	if len(os.Args) == 1 || os.Args[1] == "serv" {
		fmt.Printf("Starting teamvite server on port 8080\n")
		go func() { m.HTTPServer.Open() }()
	} else if os.Args[1] == "resetpassword" {
		// if err := ResetPassword(db, os.Args[2], os.Args[3]); err != nil {
		// 	fmt.Printf("Error: %v", err)
		// }
	} else {
		fmt.Printf("ERROR: unknown command %s\n", os.Args[1])
	}
	return nil
}

func (m *Main) Close() error {
	return nil
}
