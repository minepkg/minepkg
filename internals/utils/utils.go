package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// lineMatch matches the git output
var lineMatch = regexp.MustCompile("(.*)\r?\n?$")

// SimpleGitExec runs a git command and returns the output in a easy to process way
func SimpleGitExec(args string) (string, error) {
	splitArgs := strings.Split(args, " ")
	cmd := exec.Command("git", splitArgs...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	cleanOut := lineMatch.FindStringSubmatch(string(out))
	return cleanOut[1], err
}

// OpenBrowser opens the given url in a browser
func OpenBrowser(url string) {
	var err error

	fmt.Println("opening ", url)

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Run()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	case "darwin":
		err = exec.Command("open", url).Run()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		panic(err)
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
