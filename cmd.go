package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	datastoreRunning := make(chan struct{})
	go func() {
		cmd := exec.Command("gcloud", "beta", "emulators", "datastore", "start")
		cmd.Stderr = &watchFor{
			base:     os.Stderr,
			sentinel: "is now running",
			c:        datastoreRunning,
		}
		cmd.Stdout = os.Stdout
		if err := cmd.Start(); err != nil {
			log.Fatalf("could not start datastore emulator: %v", err)
		}
		cmd.Wait()
		close(datastoreRunning)
	}()
	<-datastoreRunning

	cmd := exec.Command("gcloud", "beta", "emulators", "datastore", "env-init")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("could not get env: %v", err)
	}
	env := strings.Split(string(out), "\n")
	for i, v := range env {
		env[i] = strings.Replace(v, "export ", "", -1)
	}
	env = append(env, os.Environ()...)

	cmd = exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Env = env
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Printf("exit code from subprocess: %q", err)
		os.Exit(1)
	}
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
	if err != nil {
		return
	}
	if r.done {
		return
	}
	n, err = r.buf.Write(data)

	if strings.Contains(r.buf.String(), r.sentinel) {
		close(r.c)
		r.done = true
	}
	return
}
