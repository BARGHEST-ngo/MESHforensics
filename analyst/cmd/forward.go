// Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// Portions of this code are derived from androidqf (Android Quick Forensics)
// Copyright (c) 2021–2022 Claudio Guarnieri.
// Use of this software is governed by the MVT License 1.1, available at:
// https://license.mvt.re/1.1/

package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/BARGHEST-ngo/MESH/analyst/session"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func ForwardCmd(s *session.Session, in *Input) *ffcli.Command {
	return &ffcli.Command{
		Name:       "forward",
		ShortUsage: "forward on|off",
		ShortHelp:  "Send a device's internet traffic through this analyst for capture",
		FlagSet:    flag.NewFlagSet("forward", flag.ContinueOnError),
		Exec: func(ctx context.Context, args []string) error {
			return runForwardCmd(ctx, args, s, in)
		},
	}
}

func runForwardCmd(ctx context.Context, args []string, s *session.Session, in *Input) error {
	if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
		return errors.New("usage: forward on|off")
	}

	on := args[0] == "on"
	dev, err := chooseDevice(ctx, s, in)
	if err != nil {
		return err
	}

	if err := s.SetForwarding(dev.ID, on); err != nil {
		return err
	}
	fmt.Fprintf(in.Output(), "requested %s to enable network forwarding %s\n", dev.Name, args[0])
	return nil
}
