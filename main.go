package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	sleepSeconds := flag.Int("s", 30, "")
	command := flag.String("c", "", "")
	flag.Parse()

	splitCommand := strings.Split(*command, " ")

	cmd := exec.Command(splitCommand[0], splitCommand[1:]...)
	printMessage(fmt.Sprintf("command to retry: %s", cmd.String()), false)

	for {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			wrappedErr := fmt.Errorf("failed at cmd.StdoutPipe: %w", err)
			printMessage(wrappedErr.Error(), true)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			wrappedErr := fmt.Errorf("failed at cmd.StderrPipe: %w", err)
			printMessage(wrappedErr.Error(), true)
		}
		p := &pipes{
			stderr,
			stdout,
		}
		doneLogging := make(chan bool, 1)
		go logCommandOutput(doneLogging, p)

		err = cmd.Start()
		if err != nil {
			wrappedErr := fmt.Errorf("failed to start underlying program: %w", err)
			printMessage(wrappedErr.Error(), true)
		}

		err = cmd.Wait()
		doneLogging <- true
		if err != nil {
			wrappedErr := fmt.Errorf("underlying program exited: %w", err)
			printMessage(wrappedErr.Error(), true)
		}

		time.Sleep(time.Duration(*sleepSeconds) * time.Second)
	}
}

func printMessage(message string, fatal bool) {
	fmt.Printf("%s%s", message, "\n")
	if fatal {
		os.Exit(1)
	}
}

type pipes struct {
	stderr io.ReadCloser
	stdout io.ReadCloser
}

func logCommandOutput(done <-chan bool, p *pipes) {
	for {
		select {
		case <-done:
			return
		default:
			stdoutOutput, err := io.ReadAll(p.stdout)
			maybeStdout := string(stdoutOutput)
			if err != nil {
				wrappedErr := fmt.Errorf("failed at stdout read: %w", err)
				printMessage(wrappedErr.Error(), false)
			}
			if len(maybeStdout) > 0 {
				message := fmt.Sprintf("new stdout: %s", maybeStdout)
				printMessage(message, false)
			}
			stderrOutput, err := io.ReadAll(p.stderr)
			maybeStderr := string(stderrOutput)
			if err != nil {
				wrappedErr := fmt.Errorf("failed at stderr read: %w", err)
				printMessage(wrappedErr.Error(), false)
			}
			if len(maybeStderr) > 0 {
				message := fmt.Sprintf("new stderr: %s", maybeStdout)
				printMessage(message, false)
			}
			time.Sleep(1 * time.Second)
		}
	}
}
