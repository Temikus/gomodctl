package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/beatlabs/gomodctl/internal/cmd/check"
	"github.com/beatlabs/gomodctl/internal/cmd/info"
	licensecmd "github.com/beatlabs/gomodctl/internal/cmd/license"
	scancmd "github.com/beatlabs/gomodctl/internal/cmd/scan"
	"github.com/beatlabs/gomodctl/internal/cmd/search"
	updatecmd "github.com/beatlabs/gomodctl/internal/cmd/update"
	"github.com/beatlabs/gomodctl/internal/godoc"
	"github.com/beatlabs/gomodctl/internal/license"
	"github.com/beatlabs/gomodctl/internal/module"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ro RootOptions
var version string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gomodctl",
	Short: "Search, Check and Update Go modules.",
	Long: `gomodctl is a Go tool that provides interactive search, check and update features for Go modules.

Example:

  gomodctl search mongo

This command will search in all public Go packages and return matching results for term "mongo".`,
	Version: version,
}

// RootOptions is exported.
type RootOptions struct {
	config   string
	registry string
	json     bool
	path     string
}

// Execute is exported.
func Execute() {
	// Since this application is a one-shot execution there is no point of making GC available and slow down the application.
	// https://www.dotconferences.com/2019/03/bryan-boreham-go-tune-your-memory
	// https://www.youtube.com/watch?v=uyifh6F_7WM
	debug.SetGCPercent(-1)

	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-signals:
			cancel()
		case <-ctx.Done():
		}
	}()

	ro.config = viper.GetString("config")
	ro.registry = viper.GetString("registry")

	gd := godoc.NewClient(ctx)
	checker := module.Checker{Ctx: ctx}
	updater := module.Updater{Ctx: ctx}
	licenseChecker, err := license.NewChecker(ctx)
	scanner := module.Scanner{Ctx: ctx}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Add sub-commands
	rootCmd.AddCommand(search.NewCmdSearch(gd))
	rootCmd.AddCommand(info.NewCmdInfo(gd))
	rootCmd.AddCommand(check.NewCmdCheck(&checker))
	rootCmd.AddCommand(updatecmd.NewCmdUpdate(&updater))
	rootCmd.AddCommand(licensecmd.NewCmdLicense(licenseChecker))
	rootCmd.AddCommand(scancmd.NewCmdScan(&scanner))

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&ro.config, "config", "", "config file (default is $HOME/gomodctl.yml)")
	rootCmd.PersistentFlags().StringVar(&ro.registry, "registry", "", "URI of the registry to be used for search")
	rootCmd.PersistentFlags().BoolVar(&ro.json, "json", false, "Print JSON result")
	rootCmd.PersistentFlags().StringVar(&ro.path, "path", "", "Optional go.mod parent directory")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("registry", rootCmd.PersistentFlags().Lookup("registry"))
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
	viper.BindPFlag("path", rootCmd.PersistentFlags().Lookup("path"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigType("yaml")

	if ro.config != "" {
		// Use config file from the flag.
		viper.SetConfigFile(ro.config)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if ro.path != "" {
			viper.AddConfigPath(ro.path)
		}
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)

		viper.SetConfigName("gomodctl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Println(err)
	}
}

func main() {
	Execute()
}
