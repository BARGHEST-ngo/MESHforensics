// Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"slices"

	"github.com/BARGHEST-ngo/MESH/analyst/session"
	meshnet "github.com/BARGHEST-ngo/MESHnet"
	"github.com/BARGHEST-ngo/androidqf_mesh/adb"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type AndroidPeer struct {
	IP       string
	HostName string
	DNSName  string
}

type PairingArgs struct {
	Host        string
	PairPort    int
	DebugPort   int
	PairingCode string
}

const maxAutoPairAttempts = 5

func AdbPairCmd(s *session.Session, in *Input) *ffcli.Command {
	fs := flag.NewFlagSet("adbpair", flag.ContinueOnError)

	return &ffcli.Command{
		Name:       "adbpair",
		ShortUsage: "adbpair",
		ShortHelp:  "Pair & connect to a device's wireless debugging via ADB",
		FlagSet:    fs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unexpected arguments: %v", args)
			}
			return runAdbPair(ctx, args, s, in)
		},
	}
}

func runAdbPair(ctx context.Context, args []string, s *session.Session, in *Input) error {
	dev, err := chooseDevice(ctx, s, in)
	if err != nil {
		return err
	}

	fmt.Printf("Using device: %s\n\n", dev.Name)
	fmt.Println("On the device:")
	fmt.Println("1. Turn on Wireless debugging")
	fmt.Println("2. Tap 'Pair device with pairing code'")

	code, err := in.ReadValid(ctx, "Pairing code: ", validatePairingCode)
	if err != nil {
		return err
	}

	pairPort, err := in.ReadPort(ctx, "Pairing port (the number after ':' under the code): ")
	if err != nil {
		return err
	}

	if err := initADB(); err != nil {
		return err
	}

	pairAddr, err := s.BridgeADB(dev.ID, pairPort)
	if err != nil {
		return err
	}
	if err := pair(pairAddr, code); err != nil {
		return err
	}

	debugPort, err := in.ReadPort(ctx, "Debug port (Wireless debugging screen, under 'IP address & Port'): ")
	if err != nil {
		return err
	}
	debugAddr, err := s.BridgeADB(dev.ID, debugPort)
	if err != nil {
		return err
	}
	if err := connect(debugAddr); err != nil {
		return err
	}
	if err := validateConnect(debugAddr); err != nil {
		return err
	}

	s.SetADBAddr(dev.ID, debugAddr)

	fmt.Println("Device connected. You may proceed with the acquisition: run 'adbcollect'")
	return nil
}

func chooseDevice(ctx context.Context, s *session.Session, in *Input) (meshnet.ClientInfo, error) {
	devices := s.Clients()
	switch len(devices) {
	case 0:
		return meshnet.ClientInfo{}, errors.New("no device connected")
	case 1:
		return devices[0], nil
	}

	names := make([]string, len(devices))
	for i, d := range devices {
		names[i] = d.Name
	}

	i, err := in.Select(ctx, "Several devices are connected:", names)
	if err != nil {
		return meshnet.ClientInfo{}, err
	}
	return devices[i], nil
}

func initADB() error {
	adbClient, err := adb.New()
	if err != nil {
		return fmt.Errorf("failed to initialise ADB: %w", err)
	}
	adb.Client = adbClient
	return nil
}

func pair(addr, code string) error {
	output, err := adb.Client.Exec("pair", addr, code)
	if err != nil {
		return fmt.Errorf("ADB pair failed: %w\nOutput: %s", err, string(output))
	}
	fmt.Println("ADB pair successful")
	return nil
}

func connect(addr string) error {
	fmt.Println("Connecting to device...")
	output, err := adb.Client.Exec("connect", addr)
	if err != nil {
		return fmt.Errorf("ADB connect failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

func validateConnect(addr string) error {
	fmt.Println("Validating ADB session...")
	devices, err := adb.Client.Devices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}
	if !slices.Contains(devices, addr) {
		return fmt.Errorf("device not connected through MESH (adb sees: %v)", devices)
	}
	return nil
}
