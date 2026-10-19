package server

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"claimconfig/component/repositories"
	"claimconfig/component/services"
	"claimconfig/utils"
)

const configPath = "config/application.yml"

// Main runs a subcommand and returns the process exit code:
//
//	serve (default)                      start the HTTP server
//	migrate [--dry-run]                  apply pending migrations
//	create-user --email --name [--role] [--locale] [--password-stdin]
func Main(args []string) int {
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	cfg, err := utils.LoadConfig(configPath)
	if err != nil {
		utils.ShowErrorLogs("load config", err)
		return 1
	}
	switch cmd {
	case "serve":
		err = serve(cfg)
	case "migrate":
		err = migrateCmd(cfg, args)
	case "create-user":
		err = createUserCmd(cfg, args)
	default:
		err = fmt.Errorf("unknown command %q (serve | migrate | create-user)", cmd)
	}
	if err != nil {
		utils.ShowErrorLogs(cmd, err)
		return 1
	}
	return 0
}

func serve(cfg utils.AppConfig) error {
	ctx := context.Background()
	app, err := Initialize(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.Close()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           app.Engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		utils.ShowInfoLogs("listening on http://localhost:%d (Swagger UI: /swagger/index.html)", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			utils.ShowErrorLogs("http server", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func migrateCmd(cfg utils.AppConfig, args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "only list applied and pending migrations")
	if err := fs.Parse(args); err != nil {
		return err
	}
	db, err := repositories.Open(cfg.Database.URL)
	if err != nil {
		return err
	}
	defer db.Close()
	return Migrate(context.Background(), db, *dryRun, os.Stdout)
}

// createUserCmd reads the password from stdin (never from an argument, so it
// stays out of shell history and process lists).
func createUserCmd(cfg utils.AppConfig, args []string) error {
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	email := fs.String("email", "", "login email (required)")
	name := fs.String("name", "", "display name (required)")
	role := fs.String("role", "viewer", "admin | operator | viewer")
	locale := fs.String("locale", "vi", "vi | en")
	fromStdin := fs.Bool("password-stdin", false, "read the password from stdin; without it the account is Google-only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	password := ""
	if *fromStdin {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return fmt.Errorf("read password: %w", err)
		}
		password = strings.TrimRight(line, "\r\n")
	}
	db, err := repositories.Open(cfg.Database.URL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := Migrate(context.Background(), db, false, os.Stdout); err != nil {
		return err
	}
	auth := services.NewAuthService(repositories.NewUserRepo(db), repositories.NewSessionRepo(db), repositories.NewRoleRepo(db), cfg.Auth)
	id, err := auth.CreateUser(context.Background(), services.CreateUserInput{
		Email: *email, Name: *name, RoleCode: *role, Password: password, Locale: *locale,
	})
	if err != nil {
		return err
	}
	utils.ShowInfoLogs("created user #%d %s (role %s)", id, strings.ToLower(*email), *role)
	return nil
}
