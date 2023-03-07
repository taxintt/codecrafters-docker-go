package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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

func copyExecutablePath(source, dest string) error {
	sourceFileStat, err := os.Stat(source)
	if err != nil {
		return err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(dest, os.O_RDWR|os.O_WRONLY, sourceFileStat.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	cmd := exec.Command(command, args...)
	setupOutput(cmd)

	// create executable path (e.g. /usr/local/bin/docker-explorer)
	// https://text.baldanders.info/golang/deprecation-of-ioutil/
	chrootDir, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Printf("error creating temp dir: %v", err)
		os.Exit(1)
	}

	if err = copyExecutableIntoDir(chrootDir, command); err != nil {
		fmt.Printf("error copying executable into chroot dir: %v", err)
		os.Exit(1)
	}

	// Create /dev/null so that cmd.Run() doesn't complain
	if err = createDevNull(chrootDir); err != nil {
		fmt.Printf("error creating /dev/null: %v", err)
		os.Exit(1)
	}

	if err = syscall.Chroot(chrootDir); err != nil {
		fmt.Printf("chroot err: %v", err)
		os.Exit(1)
	}

	// chroot
	syscall.Chroot(chrootDir)

	if err := cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func copyExecutableIntoDir(chrootDir string, executablePath string) error {
	executablePathInChrootDir := path.Join(chrootDir, executablePath)

	if err := os.MkdirAll(path.Dir(executablePathInChrootDir), 0750); err != nil {
		return err
	}

	return copyExecutablePath(executablePath, executablePathInChrootDir)
}

func createDevNull(chrootDir string) error {
	if err := os.MkdirAll(path.Join(chrootDir, "dev"), 0750); err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(chrootDir, "dev", "null"), []byte{}, 0644)
}
