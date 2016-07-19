package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type Job interface {
	Start() error
	WaitReady()
	Env() []string
	Stop() error
}

func main() {
	datastore := &Datastore{}
	if err := datastore.Start(); err != nil {
		log.Fatalf("Could not start datastore: %v", err)
	}
	pubsub := &PubSub{}
	if err := pubsub.Start(); err != nil {
		log.Fatalf("Could not start datastore: %v", err)
	}

	datastore.WaitReady()
	pubsub.WaitReady()

	env := os.Environ()
	env = append(env, datastore.Env()...)
	env = append(env, pubsub.Env()...)

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Env = env
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmdErr := cmd.Run()

	if err := datastore.Stop(); err != nil {
		log.Fatalf("Could not stop datastore: %v", err)
	}
	if err := pubsub.Stop(); err != nil {
		log.Fatalf("Could not stop datastore: %v", err)
	}
	if cmdErr != nil {
		log.Fatal(cmdErr)
	}
}

type PubSub struct {
	cmd   *exec.Cmd
	ready chan struct{}
}

func (j *PubSub) Start() error {
	if j.ready != nil {
		return errors.New("pubsub: already started")
	}
	j.ready = make(chan struct{})

	j.cmd = exec.Command("gcloud", "-q", "beta", "emulators", "pubsub", "start")
	j.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	j.cmd.Stderr = &watchFor{
		base:     os.Stderr,
		sentinel: "Server started, listening",
		c:        j.ready,
	}
	j.cmd.Stdout = os.Stdout
	return j.cmd.Start()
}

func (j *PubSub) WaitReady() {
	<-j.ready
}

func (j *PubSub) Stop() error {
	pgid, err := syscall.Getpgid(j.cmd.Process.Pid)
	if err != nil {
		return err
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		return err
	}
	j.cmd.Wait()
	return nil
}

func (j *PubSub) Env() []string {
	cmd := exec.Command("gcloud", "beta", "emulators", "pubsub", "env-init")
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

type Datastore struct {
	cmd   *exec.Cmd
	ready chan struct{}
}

func (j *Datastore) Start() error {
	if j.ready != nil {
		return errors.New("datastore: already started")
	}
	j.ready = make(chan struct{})

	j.cmd = exec.Command("gcloud", "-q", "beta", "emulators", "datastore", "start", "--no-legacy")
	j.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	j.cmd.Stderr = &watchFor{
		base:     os.Stderr,
		sentinel: "is now running",
		c:        j.ready,
	}
	j.cmd.Stdout = os.Stdout
	return j.cmd.Start()
}

func (j *Datastore) WaitReady() {
	<-j.ready
}

func (j *Datastore) Stop() error {
	pgid, err := syscall.Getpgid(j.cmd.Process.Pid)
	if err != nil {
		return err
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		return err
	}
	j.cmd.Wait()
	return nil
}

func (j *Datastore) Env() []string {
	cmd := exec.Command("gcloud", "beta", "emulators", "datastore", "env-init")
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
