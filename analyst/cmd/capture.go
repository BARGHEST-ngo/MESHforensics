// Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/BARGHEST-ngo/MESH/analyst/session"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func CaptureCmd(s *session.Session, in *Input) *ffcli.Command {
	startFS := flag.NewFlagSet("capture start", flag.ContinueOnError)
	iface := startFS.String("i", "eth0", "network interface to capture on")

	start := &ffcli.Command{
		Name:       "start",
		ShortUsage: "capture start [-i eth0]",
		ShortHelp:  "Start capturing forwarded traffic in the background",
		FlagSet:    startFS,
		Exec: func(ctx context.Context, args []string) error {
			return startCmd(ctx, s, in, iface)
		},
	}

	stop := &ffcli.Command{
		Name:       "stop",
		ShortUsage: "capture stop",
		ShortHelp:  "Stop the running capture",
		Exec: func(ctx context.Context, args []string) error {
			return stopCmd(s, in)
		},
	}

	return &ffcli.Command{
		Name:       "capture",
		ShortUsage: "capture start|stop",
		ShortHelp:  "Record a device's forwarded traffic to a pcap file",
		LongHelp: `Runs tcpdump in the background.
The pcap is written to the device's acquisition folder alongside adbcollect's output.
Only one device may forward traffic while capturing.`,
		FlagSet:     flag.NewFlagSet("capture", flag.ContinueOnError),
		Subcommands: []*ffcli.Command{start, stop},
		Exec: func(ctx context.Context, args []string) error {
			return errors.New("usage: capture start|stop")
		},
	}
}

func startCmd(ctx context.Context, s *session.Session, in *Input, iface *string) error {
	dev, err := chooseDevice(ctx, s, in)
	if err != nil {
		return err
	}
	path, err := s.StartCapture(dev.ID, *iface)
	if err != nil {
		return err
	}
	fmt.Fprintf(in.Output(), "capturing %s's traffic to %s\nrun 'forward on' if it isn't already, and 'capture stop' when done\n", dev.Name, path)
	return nil
}

func stopCmd(s *session.Session, in *Input) error {
	path, summary, err := s.StopCapture()
	if err != nil {
		return err
	}
	fmt.Fprintf(in.Output(), "capture saved to %s\n%s\n", path, summary)
	return nil
}
