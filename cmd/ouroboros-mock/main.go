// Copyright 2024 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/blinklabs-io/ouroboros-mock/internal/conversation"
	"github.com/blinklabs-io/ouroboros-mock/internal/version"

	"github.com/spf13/cobra"
)

const (
	programName = "ouroboros-mock"
)

var cmdlineFlags = struct {
	debug         bool
	listenPort    int
	listenAddr    string
	listenSocket  string
	keepListening bool
}{}

func main() {
	// Setup commandline handling
	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s [flags] <conversation file(s)>", programName),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("you must specify a conversation file")
			}
			if len(args) > 1 {
				return errors.New("you cannot specify more than one conversation file")
			}
			return nil
		},
		Run: cmdRun,
	}

	cmd.Flags().BoolVarP(&cmdlineFlags.debug, "debug", "D", false, "enable debug logging")
	cmd.Flags().IntVarP(&cmdlineFlags.listenPort, "listen-port", "p", 4000, "port to listen on")
	cmd.Flags().StringVarP(&cmdlineFlags.listenAddr, "listen-address", "a", "", "address to listen on (defaults to all addresses)")
	cmd.Flags().StringVarP(&cmdlineFlags.listenSocket, "listen-socket", "S", "", "UNIX socket path to listen on (overrides port/address)")
	cmd.Flags().BoolVarP(&cmdlineFlags.keepListening, "keep-listening", "k", false, "keep listening for additional connections after the first")

	if err := cmd.Execute(); err != nil {
		// NOTE: we purposely don't display the error, since cobra will have already displayed it
		os.Exit(1)
	}
}

func cmdRun(cmd *cobra.Command, args []string) {
	configureLogger()
	conv, err := conversation.NewFromFile(args[0])
	if err != nil {
		fmt.Printf("ERROR: failed to load conversation file: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("conv.Entries[0].Input = %#v\n", conv.Entries[0].Input)
	slog.Info(fmt.Sprintf("starting %s %s", programName, version.GetVersionString()))
	// TODO
}

func configureLogger() {
	// Configure default logger
	var logger *slog.Logger
	if cmdlineFlags.debug {
		logger = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
		)
	} else {
		logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		)
	}
	slog.SetDefault(logger)
}
