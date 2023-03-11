package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type token struct {
	token       string `json:"token"`
	accessToken string `json:"access_token"`
}

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	chrootDir, err := ioutil.TempDir("", "")
	if err := os.MkdirAll(filepath.Join(chrootDir, filepath.Dir(command)), os.ModeDir); err != nil {
		log.Fatal(fmt.Errorf("failed to create rootDir: %w", err))
	}
	defer os.RemoveAll(chrootDir)

	// copy executable file (e.g. ls)
	if err = copyExecutableFile(command, chrootDir); err != nil {
		log.Fatal(err)
	}

	// workaround for chroot
	if err := createDevNullDir(chrootDir); err != nil {
		log.Fatal(err)
	}

	if err = syscall.Chroot(chrootDir); err != nil {
		log.Fatal(fmt.Errorf("failed to execute chroot: %w", err))
	}

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID,
	}

	token, err := getBearerToken()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token) // debug

	if err := cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func copyExecutableFile(command, rootDir string) error {
	src, err := os.Open(command)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	srcInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	dst, err := os.OpenFile(filepath.Join(rootDir, command), os.O_CREATE|os.O_WRONLY, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src.Name(), dst.Name(), err)
	}

	if err := src.Close(); err != nil {
		return fmt.Errorf("failed to close source file: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("failed to close destination file: %w", err)
	}

	return nil
}

func createDevNullDir(chrootDir string) error {
	err := os.Mkdir(filepath.Join(chrootDir, "dev"), os.ModeDir)
	if err != nil {
		return fmt.Errorf("failed to create /dev directory: %w", err)
	}

	devnull, err := os.Create(filepath.Join(chrootDir, "/dev/null"))
	if err != nil {
		return fmt.Errorf("failed to create /dev/null file: %w", err)
	}

	if err := devnull.Close(); err != nil {
		return fmt.Errorf("failed to close /dev/null file: %w", err)
	}
	return nil
}

func getBearerToken() (string, error) {
	var tokens []token

	service := "registry.hub.docker.com"
	url := fmt.Sprintf(`http://auth.docker.io/token?service=%s`, service)

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to call GET http://auth.docker.io/token: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)

		if err := json.Unmarshal(body, &tokens); err != nil {
			return "", fmt.Errorf("failed to parse http response: %w", err)
		}
	}
	return "", nil
}
