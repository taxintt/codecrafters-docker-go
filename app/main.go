package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	ps, err := exec.Command(command, args...).CombinedOutput()
	if err != nil {
		// write to stdout
		fmt.Println(err.Error())
		os.Stderr.Write([]byte(err.Error()))
	}

	// write to stdout
	os.Stdout.Write(ps)
}
