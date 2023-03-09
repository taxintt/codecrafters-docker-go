package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	rootDir, err := ioutil.TempDir("", "")
	if err := os.MkdirAll(filepath.Join(rootDir, filepath.Dir(command)), os.ModeDir); err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(rootDir)

	src, err := os.Open(command)
	if err != nil {
		os.Exit(1)
	}
	srcInfo, err := src.Stat()
	if err != nil {
		os.Exit(1)
	}

	dst, err := os.OpenFile(filepath.Join(rootDir, command), os.O_CREATE|os.O_WRONLY, srcInfo.Mode())
	if err != nil {
		os.Exit(1)
	}
	if _, err := io.Copy(dst, src); err != nil {
		os.Exit(1)
	}

	src.Close()
	dst.Close()

	// workaround for chroot
	os.Mkdir(filepath.Join(rootDir, "dev"), os.ModeDir)
	devnull, _ := os.Create(filepath.Join(rootDir, "/dev/null"))
	devnull.Close()

	// if err = syscall.Chroot(rootDir); err != nil {
	// 	fmt.Printf("chroot err: %v", err)
	// 	os.Exit(1)
	// }
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:  rootDir,
		Setpgid: true,
		Pgid:    1,
	}

	if err := cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
