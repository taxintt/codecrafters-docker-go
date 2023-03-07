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

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	cmd := exec.Command(command, args...)

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

	err = os.MkdirAll(filepath.Join("mydocker", filepath.Dir(command)), os.ModeDir)
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll("mydocker")
	src, err := os.Open(command)
	if err != nil {
		os.Exit(1)
	}
	srcInfo, err := src.Stat()
	if err != nil {
		os.Exit(1)
	}

	dst, err := os.OpenFile(filepath.Join("mydocker", command), os.O_CREATE|os.O_WRONLY, srcInfo.Mode())
	if err != nil {
		os.Exit(1)
	}
	if _, err := io.Copy(dst, src); err != nil {
		os.Exit(1)
	}

	src.Close()
	dst.Close()

	// workaround for chroot
	os.Mkdir(filepath.Join("mydocker", "dev"), os.ModeDir)
	devnull, _ := os.Create(filepath.Join("mydocker", "/dev/null"))
	devnull.Close()

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
