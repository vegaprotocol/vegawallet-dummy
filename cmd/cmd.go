package cmd

import (
	"os"
)

func Execute() {
	c := NewCmdRoot()

	execErr := c.Execute()
	if execErr == nil {
		return
	}

	defer os.Exit(1)

}
