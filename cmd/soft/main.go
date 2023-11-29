package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/cmd/soft/admin"
	"github.com/charmbracelet/soft-serve/cmd/soft/browse"
	"github.com/charmbracelet/soft-serve/cmd/soft/hook"
	"github.com/charmbracelet/soft-serve/cmd/soft/repo"
	"github.com/charmbracelet/soft-serve/cmd/soft/serve"
	"github.com/charmbracelet/soft-serve/cmd/soft/settings"
	"github.com/charmbracelet/soft-serve/cmd/soft/shell"
	"github.com/charmbracelet/soft-serve/cmd/soft/user"
	"github.com/charmbracelet/soft-serve/pkg/config"
	logr "github.com/charmbracelet/soft-serve/pkg/log"
	"github.com/charmbracelet/soft-serve/pkg/version"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""

	// CommitDate contains the date of the commit that this application was
	// built against. It's set via ldflags when building.
	CommitDate = ""

	// When this flag is set, the user will be checked for access to the
	// repository before the command is run during cmd.CheckUserHasAccess.
	strict bool

	rootCmd = &cobra.Command{
		Use:          "soft",
		Short:        "A self-hostable Git server for the command line",
		Long:         "Soft Serve is a self-hostable Git server for the command line.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return browse.Command.RunE(cmd, args)
		},
	}

	manCmd = &cobra.Command{
		Use:    "man",
		Short:  "Generate man pages",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			manPage, err := mcobra.NewManPage(1, rootCmd)
			if err != nil {
				return err
			}

			manPage = manPage.WithSection("Copyright", "(C) 2021-2023 Charmbracelet, Inc.\n"+
				"Released under MIT license.")
			fmt.Println(manPage.Build(roff.NewDocument()))
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(
		admin.Command,
		browse.Command,
		hook.Command,
		manCmd,
		repo.Command,
		serve.Command,
		settings.Command,
		shell.Command,
		user.Command,
	)
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.PersistentFlags().BoolVarP(&strict, "strict", "", false, "Check if the user has access to the command")

	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
			Version = info.Main.Version
		} else {
			Version = "unknown (built from source)"
		}
	}
	rootCmd.Version = Version

	version.Version = Version
	version.CommitSHA = CommitSHA
	version.CommitDate = CommitDate
}

func main() {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	if cfg.Exist() {
		if err := cfg.Parse(); err != nil {
			log.Fatal(err)
		}
	}

	if err := cfg.ParseEnv(); err != nil {
		log.Fatal(err)
	}

	ctx = config.WithContext(ctx, cfg)
	logger, f, err := logr.NewLogger(cfg)
	if err != nil {
		log.Errorf("failed to create logger: %v", err)
	}

	ctx = log.WithContext(ctx, logger)
	if f != nil {
		defer f.Close() // nolint: errcheck
	}

	// Set global logger
	log.SetDefault(logger)

	var opts []maxprocs.Option
	if config.IsVerbose() {
		opts = append(opts, maxprocs.Logger(log.Debugf))
	}

	// Set the max number of processes to the number of CPUs
	// This is useful when running soft serve in a container
	if _, err := maxprocs.Set(opts...); err != nil {
		log.Warn("couldn't set automaxprocs", "error", err)
	}

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}