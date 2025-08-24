package cmd

import (
	"ask/ask_db"
	structs "ask/internal/model"
	"bytes"
	"fmt"
	"os"
	"strings"

	"slices"

	"github.com/abiosoft/ishell/v2"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	consoleCmd = &cobra.Command{
		Use:     "console",
		Short:   "Start the ask console",
		Aliases: []string{"ls"},
		Long: `Start the ask console, which was built to feel like the metasploit console to search for snippets, set variable values, and render them.
`,
		Run: func(_ *cobra.Command, args []string) {
			askConsole()
		},
	}
)

func askConsole() {
	shell := ishell.New()

	shell.Println("ask interactive shell")

	db := ask_db.DB
	shell.AddCmd(&ishell.Cmd{
		Name: "search",
		Help: "search for an existing snippet",
		Func: func(c *ishell.Context) {
			var searchTerm string
			if len(c.Args) == 0 {
				searchTerm = "*"
			} else {
				// Need to fix this search string to allow any number of arguments
				searchTerm = c.Args[0]
			}
			err := browse(searchTerm, ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Ran into error executing browse function: %w", err)
			}
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "use",
		Help: "use an existing snippet",
		Completer: func([]string) []string {
			var snippetNames []string

			stmt, _ := db.Prepare("SELECT name FROM snippets")
			rows, _ := stmt.Query()

			var snippetName string
			for rows.Next() {
				rows.Scan(&snippetName)

				if !slices.Contains(snippetNames, snippetName) {
					snippetNames = append(snippetNames, snippetName)
				}

			}
			return snippetNames
		},
		Func: func(c *ishell.Context) {
			useConsole(c.Args[0], shell)
		},
	})

	shell.Run()
}

func setVariables(variables map[string]*structs.Variable) (variableMap map[string]string) {

	variableMap = make(map[string]string)
	for variable, variableStruct := range variables {
		variableMap[variable] = variableStruct.DefaultValue
		if variableStruct.DefaultValue == "" {
			variableMap[variable] = "{{ " + variable + "_NOT_SET }}"
		}
	}
	return variableMap
}

func useConsole(snippetString string, shell *ishell.Shell) error {
	db := ask_db.DB
	snippetID, snippetName, _ := getSnippetID(snippetString, db)
	sub := ishell.New()
	promptString := fmt.Sprintf("%s >", snippetName)
	sub.SetPrompt(promptString)

	//Grab all the snippet data
	snippetData, _ := getSnippetData(snippetID, db)

	//Populate variables
	variableMap := setVariables(snippetData.Variables)

	sub.AddCmd(&ishell.Cmd{
		Name: "show",
		Help: "Show information about a snippet",
		Func: func(sc *ishell.Context) {

			var headerTable bytes.Buffer
			tw := table.NewWriter()
			tw.SetOutputMirror(&headerTable)
			tw.AppendHeader(table.Row{"Name", "Description"})
			tw.AppendRow(table.Row{snippetData.Name, snippetData.Description})
			tw.SetStyle(table.StyleRounded)
			tw.Render()

			// Merge everything into headerSection
			sc.Print(headerTable.String())

			//show variables

			var variableTable bytes.Buffer
			tw = table.NewWriter()
			tw.SetOutputMirror(&variableTable)
			tw.AppendHeader(table.Row{"Variable Name", "Value", "Example Value", "Description"})

			var variableKey []string
			for variable, variableInfo := range snippetData.Variables {
				variableKey = append(variableKey, variable)
				tw.AppendRow(table.Row{variable, variableMap[variable], variableInfo.ExampleValue, variableInfo.Description})
			}
			tw.SetStyle(table.StyleRounded)
			tw.Render()
			sc.Print(variableTable.String())

		},
	})
	sub.AddCmd(&ishell.Cmd{
		Name: "set",
		Help: "set variables",
		Completer: func(args []string) []string {
			if len(args) < 1 {
				var variables []string
				for variable := range snippetData.Variables {
					variables = append(variables, variable)
				}
				return variables
			} else {
				//in the future, might suggest default or multichoice
				return nil
			}
		},

		Func: func(sc *ishell.Context) {

			if _, ok := variableMap[sc.Args[0]]; !ok {
				sc.Printf("%s is not a valid variable\n", sc.Args[0])
				return
			}
			variableMap[sc.Args[0]] = sc.Args[1]
		},
	})
	/* do this later
	sub.AddCmd(&ishell.Cmd{
		Name: "setg",
		Help: "set global variables",
		Func: func(sc *ishell.Context) {
			sc.Println("tab completition of variables")
		},
	})
	*/

	sub.AddCmd(&ishell.Cmd{
		Name:    "render",
		Help:    "Render the snippet",
		Aliases: []string{"run", "exploit"},
		Func: func(sc *ishell.Context) {
			//take flags for render to clipboard
			//take flags for render to file

			snippetText := snippetData.SnippetText

			for variable, value := range variableMap {
				snippetText = strings.ReplaceAll(snippetText, "{{ "+variable+" }}", value)
			}

			sc.Printf(snippetText + "\n")
		},
	})
	sub.AddCmd(&ishell.Cmd{
		Name: "back",
		Help: "Drop out of use shell",
		Func: func(sc *ishell.Context) {
			sub.Close()
		},
	})
	sub.AddCmd(&ishell.Cmd{
		Name: "search",
		Help: "Search for snippets",
		Func: func(sc *ishell.Context) {
			var searchTerm string
			if len(sc.Args) == 0 {
				searchTerm = "*"
			} else {
				// Need to fix this search string to allow any number of arguments
				searchTerm = sc.Args[0]
			}
			err := browse(searchTerm, ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Ran into error executing browse function: %w", err)
			}
		},
	})

	sub.AddCmd(&ishell.Cmd{
		Name: "use",
		Help: "use an existing snippet",
		Completer: func([]string) []string {
			var snippetNames []string

			stmt, _ := db.Prepare("SELECT name FROM snippets")
			rows, _ := stmt.Query()

			var snippetName string
			for rows.Next() {
				rows.Scan(&snippetName)
				snippetNames = append(snippetNames, snippetName)
			}
			return snippetNames
		},
		Func: func(sc *ishell.Context) {
			useConsole(sc.Args[0], shell)
		},
	})

	sub.AddCmd(&ishell.Cmd{
		Name: "exit",
		Help: "Exit the ask console",
		Func: func(sc *ishell.Context) {
			sub.Close()
			shell.Close()
		},
	})

	sub.Run()
	return nil
}
