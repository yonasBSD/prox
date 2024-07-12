package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fgrosse/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	StatusFailedProcess = 1
	StatusBadEnvFile    = 2
	StatusBadProcFile   = 3
	StatusMissingArgs   = 4

	DefaultSocketPath = ".prox.sock" // hidden file in current PWD
)

var logger *zap.Logger

var cmd = &cobra.Command{
	Use:   "prox",
	Short: "A process runner for Procfile-based applications",
	Long: fmt.Sprintf(`A process runner for Procfile-based applications

Version: %s

`, version),
	Run: run,
}

func main() {
	cmd.PersistentFlags().BoolP("verbose", "v", false, "enable detailed log output for debugging")
	cmd.Flags().AddFlagSet(startCmd.Flags())

	viper.AutomaticEnv()
	viper.SetEnvPrefix("PROX")
	viper.BindPFlags(cmd.PersistentFlags())

	cobra.OnInitialize(func() {
		debug := viper.GetBool("verbose")
		logger = prox.NewLogger(os.Stderr, debug)
	})

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cliContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGALRM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	return ctx
}

func environment(path string) (prox.Environment, error) {
	if path == "" {
		return prox.Environment{}, errors.New("env file path cannot be empty")
	}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		logger.Debug("Did not find env file. Using system env instead", zap.String("path", path))
		return prox.SystemEnv(), nil
	}

	if err != nil {
		return prox.Environment{}, errors.New("failed to open env file: " + err.Error())
	}

	logger.Debug("Reading env file", zap.String("path", path))

	defer f.Close()
	env := prox.SystemEnv()
	err = env.ParseEnvFile(f)
	return env, err
}

func processes(env prox.Environment, path string) ([]prox.Process, error) {
	var parser func(io.Reader, prox.Environment) ([]prox.Process, error)

	switch {
	case path == "" && fileExists("Proxfile"):
		logger.Debug("Reading processes from Proxfile")
		path = "Proxfile"
		parser = prox.ParseProxFile
	case path == "" && fileExists("Procfile"):
		logger.Debug("Reading processes from Procfile")
		path = "Procfile"
		parser = prox.ParseProcFile
	case path != "":
		logger.Debug("Reading processes from file specified via --procfile", zap.String("path", path))
		if strings.HasPrefix(filepath.Base(path), "Procfile") {
			parser = prox.ParseProcFile
		} else {
			parser = prox.ParseProxFile
		}
	default:
		return nil, errors.New("no Proxfile or Procfile file found. Please specify a path with --procfile")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open process file: %w", err)
	}

	defer f.Close()
	return parser(f, env)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
