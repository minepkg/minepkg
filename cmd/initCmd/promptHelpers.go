package initCmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
)

func selectPrompt(prompt *promptui.Select) string {
	_, res, err := prompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(1)
	}
	return res
}

func stringPrompt(prompt *promptui.Prompt) string {
	res, err := prompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(1)
	}
	return res
}

func boolPrompt(prompt *promptui.Prompt) bool {
	_, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			fmt.Println("Aborting")
			os.Exit(1)
		}
		return false
	}
	return true
}
