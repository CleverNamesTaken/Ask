package cmd

import (
	"fmt"
	"io"

	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

const (
	shortUsage = "Console for interacting with ask snippets"
)

var (
	consoleCmd = &cobra.Command{
		Use:   "console",
		Short: "Start ask console",
		Long:  `Start the ask console to interact with ask snippets.`,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			ask_console()
		},
	}
)

func ask_console() {
	app := console.New("ask_console")

	app.NewlineBefore = true
	app.NewlineAfter = true

	app.SetPrintLogo(func(_ *console.Console) {
		fmt.Print("BANNER GOES HERE\n")
	})

	menu := app.ActiveMenu()
	setupPrompt(menu)

	// All menus currently each have a distinct, in-memory history source.
	// Replace the main (current) menu's history with one writing to our
	// application history file. The default history is named after its menu.

	//TODO the history file ought to be at ~/.config/ask/console_history
	hist, _ := embeddedHistory(".example-history")
	menu.AddHistorySource("local history", hist)

	menu.AddInterrupt(io.EOF, exitCtrlD)

	menu.SetCommands(mainMenuCommands(app))

	// Create another menu, different from the main one.
	// It will have its own command tree, prompt engine, history sources, etc.
	clientMenu := app.NewMenu("search")
	clientMenu.AddInterrupt(io.EOF, errorCtrlSwitchMenu)
	// Add some commands to our client menu.
	// This is an example of binding "traditionally defined" cobra.Commands.
	clientMenu.SetCommands(makeClientCommands(app))
	app.Start()
}
