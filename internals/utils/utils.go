package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type simpleGitExecOutput []string

func (s simpleGitExecOutput) String() string {
	return strings.Join(s, "\n")
}

// Last returns the last line of the output (and trims whitespace!)
func (s simpleGitExecOutput) Last() string {
	return strings.TrimSpace(s[len(s)-1])
}


// SimpleGitExec runs a git command and returns the output in a easy to process way
func SimpleGitExec(args string) (simpleGitExecOutput, error) {
	splitArgs := strings.Split(args, " ")
	cmd := exec.Command("git", splitArgs...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	splitOut := strings.Split(string(out), "\n")
	cleanOut := simpleGitExecOutput(splitOut[:len(splitOut)-1])
	return cleanOut, err
}

// OpenBrowser opens the given url in a browser
func OpenBrowser(url string) {
	var err error

	fmt.Println("Opening ", url)

	// 15 seconds timeout to open the browser
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "linux":
		err = exec.CommandContext(ctx, "xdg-open", url).Run()
	case "windows":
		err = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url).Run()
	case "darwin":
		err = exec.CommandContext(ctx, "open", url).Run()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Could not open browser, please open the following url manually:")
			fmt.Println(url)
		} else {
			fmt.Println("Could not open browser:")
			panic(err)
		}
	}
}

// ReadJSONFile parses the given file into i
func ReadJSONFile(filename string, i interface{}) error {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, i)
}
