package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func setupOutput(cmd *exec.Cmd) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
}

func createDevNull() {
	// workaround for chroot
	os.Mkdir(filepath.Join("mydocker", "dev"), os.ModeDir)
	devnull, _ := os.Create(filepath.Join("mydocker", "/dev/null"))
	devnull.Close()
}

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	cmd := exec.Command(command, args...)
	setupOutput(cmd)

	if err := os.MkdirAll(filepath.Join("mydocker", filepath.Dir(command)), os.ModeDir); err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll("mydocker")

	createDevNull()

	// chroot
	syscall.Chroot(args[1])

	if err := cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
