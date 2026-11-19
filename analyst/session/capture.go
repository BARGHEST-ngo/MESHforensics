// Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrCapturing       = errors.New("a capture is already running")
	ErrNotCapturing    = errors.New("no capture is running")
	ErrOtherForwarding = errors.New("another device is forwarding")
)

type capture struct {
	cmd      *exec.Cmd
	clientID string
	path     string
	stderr   *bytes.Buffer // tcpdump's errors and packet counts
	exited   chan struct{}
	err      error
}

func (s *Session) StartCapture(clientID, iface string) (string, error) {
	s.mu.Lock()
	if s.capture != nil {
		s.mu.Unlock()
		return "", ErrCapturing
	}
	for id, d := range s.devices {
		if id != clientID && d.forwarding {
			s.mu.Unlock()
			return "", ErrOtherForwarding
		}
	}
	s.mu.Unlock()

	dir, err := s.AcquisitionDir(clientID)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, "capture-"+time.Now().UTC().Format("2006-01-02T150405Z")+".pcap")
	cmd := exec.Command("tcpdump", "-i", iface, "-n", "-U", "-w", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start tcpdump: %w", err)
	}

	c := &capture{
		clientID: clientID,
		cmd:      cmd,
		path:     path,
		stderr:   &stderr,
		exited:   make(chan struct{}),
	}
	go func() {
		c.err = cmd.Wait()
		close(c.exited)
	}()

	select {
	case <-c.exited:
		return "", fmt.Errorf("tcpdump: %v: %s", c.err, strings.TrimSpace(stderr.String()))
	case <-time.After(500 * time.Millisecond):
	}

	s.mu.Lock()
	s.capture = c
	s.mu.Unlock()
	return path, nil
}

func (s *Session) StopCapture() (path, summary string, err error) {
	s.mu.Lock()
	c := s.capture
	s.capture = nil
	s.mu.Unlock()
	if c == nil {
		return "", "", ErrNotCapturing
	}

	c.cmd.Process.Signal(os.Interrupt)
	select {
	case <-c.exited:
	case <-time.After(5 * time.Second):
		c.cmd.Process.Kill()
		<-c.exited
	}
	return c.path, strings.TrimSpace(c.stderr.String()), nil
}

func (s *Session) CapturePath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture == nil {
		return ""
	}
	return s.capture.path
}
