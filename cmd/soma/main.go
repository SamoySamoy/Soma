// Command soma runs the Soma server and its maintenance tasks.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/SamoySamoy/Soma/internal/buildinfo"
	"github.com/SamoySamoy/Soma/internal/modules/people"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/config"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	somalog "github.com/SamoySamoy/Soma/internal/platform/log"
	"github.com/SamoySamoy/Soma/internal/server"
	"github.com/SamoySamoy/Soma/internal/space"
	"github.com/SamoySamoy/Soma/migrations"
	"github.com/SamoySamoy/Soma/web"
)

const usage = `Usage: soma <command>

Commands:
  serve                 Run the HTTP server (default)
  migrate up            Apply pending migrations
  migrate down-to <v>   Roll back to migration version v
  migrate status        List migrations and whether they are applied
  version               Print the version
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	switch cmd {
	case "version":
		fmt.Println(buildinfo.Version)
		return 0
	case "help", "-h", "--help":
		fmt.Print(usage)
		return 0
	case "serve", "migrate":
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error:\n%v\n", err)
		return 1
	}
	logger := somalog.New(os.Stderr, cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cmd == "migrate" {
		err = migrate(ctx, cfg, logger, args)
	} else {
		err = serve(ctx, cfg, logger)
	}
	if err != nil {
		logger.Error("exiting with error", "error", err)
		return 1
	}
	return 0
}

func migrate(ctx context.Context, cfg config.Config, logger *slog.Logger, args []string) error {
	if len(args) == 0 {
		return errors.New("migrate needs a subcommand: up, down-to or status")
	}
	m, err := db.NewMigrator(cfg.MigrateDatabaseURL, migrations.FS())
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	switch args[0] {
	case "up":
		return m.Up(ctx, logger)
	case "down-to":
		if len(args) < 2 {
			return errors.New("migrate down-to needs a version")
		}
		v, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parse version: %w", err)
		}
		return m.DownTo(ctx, v)
	case "status":
		st, err := m.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range st {
			fmt.Printf("%-16d %-8s %s\n", s.Source.Version, s.State, s.Source.Path)
		}
		return nil
	default:
		return fmt.Errorf("unknown migrate subcommand %q", args[0])
	}
}

func serve(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	if cfg.AutoMigrate {
		m, err := db.NewMigrator(cfg.MigrateDatabaseURL, migrations.FS())
		if err != nil {
			return err
		}
		err = m.Up(ctx, logger)
		_ = m.Close()
		if err != nil {
			return err
		}
	}

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	schema, err := migrations.Latest()
	if err != nil {
		return err
	}
	clk := clock.System{}
	// Until sign-in exists, the server acts as one implicit owner (ADR-014).
	if err := space.EnsureLocal(ctx, database, clk); err != nil {
		return err
	}
	handler, err := server.New(server.Deps{
		Config:        cfg,
		Logger:        logger,
		Clock:         clk,
		Ready:         database,
		SchemaVersion: schema,
		Web:           web.Dist(),
		People:        people.NewService(database, clk),
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.HTTPAddr, "mode", cfg.Mode, "version", buildinfo.Version)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
