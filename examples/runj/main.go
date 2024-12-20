package main

import (
	"context"
	"fmt"
	"log"
	"os"

	v8 "github.com/zeiss/v8go"

	"github.com/spf13/cobra"
)

type Config struct {
	Entrypoint string
}

var cfg = &Config{}

const (
	versionFmt = "%s (%s %s)"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&cfg.Entrypoint, "entrypoint", "e", cfg.Entrypoint, "entrypoint")

	RootCmd.SilenceErrors = true
}

func initConfig() {}

var RootCmd = &cobra.Command{
	Use:   "runj",
	Short: "runj",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRoot(cmd.Context())
	},
	Version: fmt.Sprintf(versionFmt, version, commit, date),
}

func runRoot(ctx context.Context) error {
	src, err := os.ReadFile(cfg.Entrypoint)
	if err != nil {
		return err
	}

	c := v8.NewContext()
	defer c.Close()

	_, err = c.RunScript(string(src), "main.js")
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
