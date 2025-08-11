package cmd

import (
	"fmt"
	"os"

	"github.com/reeflective/console"
)

func exitCtrlD(c *console.Console) {
	os.Exit(0)

}

func switchMenu(c *console.Console) {
	fmt.Println("Switching to client menu")
	c.SwitchMenu("client")
}
