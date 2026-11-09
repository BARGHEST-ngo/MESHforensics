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
	"os"
	"path/filepath"
	"time"

	"github.com/BARGHEST-ngo/MESH/analyst/session"
	"github.com/BARGHEST-ngo/androidqf_mesh/acquisition"
	"github.com/BARGHEST-ngo/androidqf_mesh/adb"
	"github.com/BARGHEST-ngo/androidqf_mesh/log"
	"github.com/BARGHEST-ngo/androidqf_mesh/modules"
	"github.com/BARGHEST-ngo/androidqf_mesh/utils"
	"github.com/google/uuid"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var adbcollectArgs struct {
	verbose bool
	fast    string
	list    string
	module  string
	output  string
	version bool
}

func AdbcollectCmd(s *session.Session, in *Input) *ffcli.Command {
	fs := flag.NewFlagSet("adbcollect", flag.ContinueOnError)
	fs.BoolVar(&adbcollectArgs.verbose, "verbose", false, "Enable verbose output")
	fs.StringVar(&adbcollectArgs.fast, "fast", "", "Fast mode (skip some checks)")
	fs.StringVar(&adbcollectArgs.list, "list", "", "List available modules")
	fs.StringVar(&adbcollectArgs.module, "module", "", "Specific module to run")
	fs.StringVar(&adbcollectArgs.output, "output", "", "Output directory for collected data")
	fs.BoolVar(&adbcollectArgs.version, "version", false, "Show version information")

	return &ffcli.Command{
		Name:       "adbcollect",
		ShortUsage: "adbcollect [flags]",
		ShortHelp:  "Collects ADB data, using AndroidQF and WARD",
		LongHelp: `The adbcollect command initiates ADB acquisition using AndroidQF and WARD libraries. The output is in the same format ready for WARD or MVT usage.

Examples:
  adbcollect
  adbcollect --output /path/to/output
  adbcollect --module BackupTar
`,
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
			return runcollectCmd(ctx, args, s, in)
		},
	}
}

func runcollectCmd(ctx context.Context, args []string, s *session.Session, in *Input) error {
	//(ov) my experience is generally it shouldn't take more than 60 minutes.
	//we should verify this with user feedback
	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %v", args)
	}

	logLvl := log.Get().LogLevel
	if adbcollectArgs.verbose {
		log.SetLogLevel(log.DEBUG)
		defer log.SetLogLevel(logLvl)
	}

	if adbcollectArgs.version {
		log.Infof("AndroidQF version: %s", utils.Version)
		// include WARD version
		return nil
	}

	if adbcollectArgs.list != "" {
		mods := modules.List()
		log.Info("List of modules:")
		// include WARD modules if not combined with AndroidQF
		for _, mod := range mods {
			log.Infof("- %s", mod.Name())
		}
		return nil
	}

	dev, err := chooseDevice(ctx, s, in)
	if err != nil {
		return err
	}

	addr := s.ADBAddr(dev.ID)
	if addr == "" {
		return errors.New("device not connected over ADB, run adbpair")
	}
	if err := initADB(); err != nil {
		return err
	}

	return collect(ctx, addr, collectOptions{
		output: adbcollectArgs.output,
		module: adbcollectArgs.module,
		fast:   adbcollectArgs.fast != "",
	})
}

type collectOptions struct {
	output string // "" means a new folder in the working directory
	module string // "" means all modules
	fast   bool
}

func collect(ctx context.Context, addr string, opts collectOptions) error {
	adb.Client.Serial = addr

	for attempt := 1; ; attempt++ {
		var err error
		if _, err = adb.Client.GetState(); err == nil {
			break
		}
		if attempt == 3 {
			return fmt.Errorf("device not reachable over ADB, run adbpair: %w", err)
		}
		log.Debug(err)
		log.Error("Unable to get device state. Please make sure it is connected and authorized. Trying again in 5 seconds...")

		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	outputPath := opts.output
	if outputPath == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("find working directory: %w", err)
		}
		outputPath = filepath.Join(workingDirectory, uuid.New().String())
	}

	acq, err := acquisition.New(outputPath)
	if err != nil {
		return fmt.Errorf("initialise acquisition: %w", err)
	}

	// Start acquisitions
	log.Info(fmt.Sprintf("Started new acquisition in %s", acq.StoragePath))

	mods := modules.List()
	for _, mod := range mods {
		if (opts.module != "") && (opts.module != mod.Name()) {
			continue
		}
		err = mod.InitStorage(acq.StoragePath)
		if err != nil {
			log.Infof(
				"ERROR: failed to initialize storage for module %s: %v",
				mod.Name(),
				err,
			)
			continue
		}

		err = mod.Run(acq, opts.fast)
		if err != nil {
			log.Infof("ERROR: failed to run module %s: %v", mod.Name(), err)
		}
	}

	if acq.StreamingMode {
		// In streaming mode, all data is already encrypted in the zip stream
		log.Info("Finalizing encrypted acquisition...")
	} else {
		// Traditional mode: hash files, then encrypt if key exists
		err = acq.HashFiles()
		if err != nil {
			log.ErrorExc("Failed to generate list of file hashes", err)
			return err
		}

		acq.StoreInfo()

		err = acq.StoreSecurely()
		if err != nil {
			log.ErrorExc("Something failed while encrypting the acquisition", err)
			log.Warning("WARNING: The secure storage of the acquisition folder failed! The data is unencrypted!")
		}
	}

	acq.Complete()
	log.Info("Acquisition completed.")
	return nil
}
