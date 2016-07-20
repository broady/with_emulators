// Copyright 2016 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

var verbose = flag.Bool("v", false, "Pipe stdout/stderr from emulators")

func main() {
	flag.Parse()

	if err := syscall.Setpgid(os.Getpid(), os.Getpid()); err != nil {
		log.Fatalf("setpgid: %v", err)
	}
	forwardSignals()

	datastore := &Emulator{
		Command:       []string{"gcloud", "-q", "beta", "emulators", "pubsub", "start"},
		EnvCommand:    []string{"gcloud", "-q", "beta", "emulators", "pubsub", "env-init"},
		ReadySentinel: "Server started, listening",
	}
	if err := datastore.Start(); err != nil {
		log.Fatalf("Could not start datastore: %v", err)
	}

	pubsub := &Emulator{
		Command:       []string{"gcloud", "-q", "beta", "emulators", "datastore", "start", "--no-legacy"},
		EnvCommand:    []string{"gcloud", "-q", "beta", "emulators", "datastore", "env-init"},
		ReadySentinel: "is now running",
	}
	if err := pubsub.Start(); err != nil {
		log.Fatalf("Could not start pubsub: %v", err)
	}

	datastore.WaitReady()
	pubsub.WaitReady()

	env := os.Environ()
	env = append(env, datastore.Env()...)
	env = append(env, pubsub.Env()...)

	cmd := exec.Command(flag.Args()[0], flag.Args()[1:]...)
	cmd.SysProcAttr = sysprocattr()
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmdErr := cmd.Run()

	if err := datastore.Stop(); err != nil {
		log.Fatalf("Could not stop datastore: %v", err)
	}
	if err := pubsub.Stop(); err != nil {
		log.Fatalf("Could not stop pubsub: %v", err)
	}
	if cmdErr != nil {
		log.Fatal(cmdErr)
	}
}

func sysprocattr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    os.Getpid(),
	}
}

type Emulator struct {
	cmd   *exec.Cmd
	ready chan struct{}

	Command       []string
	EnvCommand    []string
	ReadySentinel string
}

func (e *Emulator) Start() error {
	if e.ready != nil {
		return errors.New("already started")
	}
	e.ready = make(chan struct{})

	e.cmd = exec.Command(e.Command[0], e.Command[1:]...)
	e.cmd.SysProcAttr = sysprocattr()
	out := ioutil.Discard
	if *verbose {
		out = os.Stderr
	}
	e.cmd.Stderr = &watchFor{
		base:     out,
		sentinel: e.ReadySentinel,
		c:        e.ready,
	}
	if *verbose {
		e.cmd.Stdout = os.Stdout
	}
	return e.cmd.Start()
}

func (e *Emulator) WaitReady() {
	<-e.ready
}

func (e *Emulator) Stop() error {
	if err := syscall.Kill(-os.Getpid(), syscall.SIGTERM); err != nil {
		return err
	}
	e.cmd.Wait()
	return nil
}

func (e *Emulator) Env() []string {
	cmd := exec.Command(e.EnvCommand[0], e.EnvCommand[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("could not get env: %v", err)
	}
	env := strings.Split(string(out), "\n")
	for i, v := range env {
		env[i] = strings.Replace(v, "export ", "", -1)
	}
	return env
}

type watchFor struct {
	base     io.Writer
	buf      bytes.Buffer
	sentinel string
	c        chan struct{}
	done     bool
}

func (r *watchFor) Write(data []byte) (n int, err error) {
	n, err = r.base.Write(data)
	if r.done || err != nil {
		return
	}

	n, err = r.buf.Write(data)
	if strings.Contains(r.buf.String(), r.sentinel) {
		close(r.c)
		r.done = true
	}
	return
}

func forwardSignals() {
	pgroup, err := os.FindProcess(-os.Getpid())
	if err != nil {
		log.Fatalf("Could not find proc 0: %v", err)
	}
	sigch := make(chan os.Signal, 1)
	go func() {
		select {
		case sig := <-sigch:
			// Forward the signal.
			pgroup.Signal(sig)
		}
	}()
	signal.Notify(sigch,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
}
