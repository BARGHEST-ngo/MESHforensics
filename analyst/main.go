// Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"

	"github.com/BARGHEST-ngo/MESH/analyst/cmd"
	"github.com/BARGHEST-ngo/MESH/analyst/session"
	"github.com/BARGHEST-ngo/androidqf_mesh/utils"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/skip2/go-qrcode"
)

func main() {
	setAndroidQFVersion()

	// Each command gets its own interrupt context (runCommand)
	// Catch Ctrl+C so it can't kill the analyst and drop every connected device
	signal.Notify(make(chan os.Signal, 1), os.Interrupt)

	in := cmd.NewInput(os.Stdin)
	s := session.New(in.Output())
	addr, err := s.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "start: %v\n", err)
		os.Exit(1)
	}
	defer s.Stop()

	qr, err := qrcode.New(addr, qrcode.Low)
	if err != nil {
		fmt.Fprintf(os.Stderr, "qr code: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(qr.ToSmallString(false))
	fmt.Printf("MESH address: %s\n\nRun 'pin' to pair a device\n", addr)

	for {
		line, err := in.ReadLine(context.Background(), "meshcli> ")
		switch {
		case errors.Is(err, cmd.ErrInterrupted):
			continue // Ctrl-C at the prompt: new prompt, like a shell
		case err != nil:
			return // Ctrl-D or end of input
		}

		args := strings.Fields(line)
		switch {
		case len(args) == 0:
			continue
		case args[0] == "quit" || args[0] == "exit":
			return
		}

		err = runCommand(s, in, args)
		switch {
		case err == nil, errors.Is(err, flag.ErrHelp):
		case errors.Is(err, cmd.ErrInterrupted), errors.Is(err, context.Canceled):
			fmt.Fprintln(os.Stderr, "cancelled")
		default:
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}
}

func runCommand(s *session.Session, in *cmd.Input, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return commands(s, in).ParseAndRun(ctx, args)
}

func commands(s *session.Session, in *cmd.Input) *ffcli.Command {
	root := &ffcli.Command{
		Name:       "meshcli",
		ShortUsage: "<command> [flags]",
		FlagSet:    flag.NewFlagSet("meshcli", flag.ContinueOnError),
		Subcommands: []*ffcli.Command{
			cmd.PinCmd(s),
			cmd.AdbPairCmd(s, in),
			cmd.AdbcollectCmd(s, in),
			cmd.AdbdisableCmd(s, in),
			cmd.AdbcleanCmd(s, in),
			cmd.StatusCmd(s),
			cmd.DisconnectCmd(s, in),
			cmd.ForwardCmd(s, in),
			cmd.CaptureCmd(s, in),
		},
		Exec: func(ctx context.Context, args []string) error {
			fmt.Fprintf(os.Stderr, "Run 'meshcli help' for usage.\n")
			return flag.ErrHelp
		},
	}

	root.Exec = func(ctx context.Context, args []string) error {
		fmt.Println(ffcli.DefaultUsageFunc(root))
		return nil
	}

	for _, c := range root.Subcommands {
		if c.FlagSet == nil {
			c.FlagSet = flag.NewFlagSet(c.Name, flag.ContinueOnError)
		}
	}
	return root
}

// androidqf's own releases set utils.Version with -ldflags, MESHforensics does not do this
// take it from the module version, acquisitions record it in their metadata
func setAndroidQFVersion() {
	if utils.Version != "" {
		return
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	for _, m := range info.Deps {
		if m.Path == "github.com/BARGHEST-ngo/androidqf_mesh" {
			utils.Version = m.Version
			return
		}
	}
}
