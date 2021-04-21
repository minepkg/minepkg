package utils

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
)

func SelectPrompt(prompt *promptui.Select) string {
	_, res, err := prompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(1)
	}
	return res
}

func StringPrompt(prompt *promptui.Prompt) string {
	res, err := prompt.Run()
	if err != nil {
		fmt.Println("Aborting")
		os.Exit(1)
	}
	return res
}

func BoolPrompt(prompt *promptui.Prompt) bool {
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
