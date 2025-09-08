package cmd

import (
	"ask/ask_db"
	structs "ask/internal/model"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"slices"

	"github.com/spf13/pflag"
	//"golang.design/x/clipboard"

	"github.com/abiosoft/ishell/v2"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var (
	consoleCmd = &cobra.Command{
		Use:   "console",
		Short: "Start the ask console",
		Long: `Start the ask console, which was built to feel like the metasploit console to search for snippets, set variable values, and render them.
`,
		Run: func(_ *cobra.Command, args []string) {
			askConsole()
		},
	}
)

func systemCommand(c *ishell.Context) {
	if c.Args[0] == "cd" {
		dir := c.Args[1]
		err := os.Chdir(dir)
		if err != nil {
			c.Println("Error changing directory:", err)

		}
		return
	}

	cmd := exec.Command("bash", "-c", strings.Join(c.Args, " "))

	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
	}
	c.Println(string(out))
	return

}

func askConsole() {
	shell := ishell.New()

	shell.Println("ask interactive shell")

	db := ask_db.DB

	shell.AddCmd(&ishell.Cmd{
		Name: "!",
		Help: "Execute a shell command",
		Func: func(c *ishell.Context) {
			//need to add a catch to allow !<cmd> to also work
			systemCommand(c)
		},
	})

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
			err := browse(searchTerm, ask_db.DB, SearchField)
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
	snippetVersion, _ := getVersion(snippetID, db)
	sub := ishell.New()
	promptString := fmt.Sprintf("%s_v%s >", snippetName, snippetVersion)
	sub.SetPrompt(promptString)

	//Grab all the snippet data
	snippetData, _ := getSnippetData(snippetID, db)

	//Populate variables
	variableMap := setVariables(snippetData.Variables)

	sub.AddCmd(&ishell.Cmd{
		Name: "!",
		Help: "Execute a shell command",
		Func: func(sc *ishell.Context) {
			//need to add a catch to allow !<cmd> to also work
			systemCommand(sc)
		},
	})

	sub.AddCmd(&ishell.Cmd{
		Name:    "show",
		Help:    "Show information about a snippet",
		Aliases: []string{"info"},
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

			if len(snippetData.Variables) > 0 {
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
			} else {
				sc.Print("\n\nNo variables for this snippet.\n\n")
			}

		},
	})
	sub.AddCmd(&ishell.Cmd{
		Name: "set",
		Help: "set variables",
		Completer: func(args []string) []string {
			var variables []string
			for variable := range snippetData.Variables {
				variables = append(variables, variable)
			}
			if len(args) == 0 {
				return variables
			} else if len(args) == 1 {
				defaultValue := snippetData.Variables[args[0]].DefaultValue
				if defaultValue != "" && !strings.Contains(defaultValue, "|") {
					return []string{defaultValue}
				} else if defaultValue != "" {
					return strings.Split(defaultValue, "|")
				}
			}
			return nil
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
		Name: "render",
		Help: `Render the snippet with the variables replaced.  This command currently supports two flags:

		--output, -o,		Save the rendered snippet to the designated file.
		--quiet,-q,		Do not render the output to stdout.  If --output is not used, then there is no point in using this flag.

		Aliases: run, exploit, r

		`,

		//--clip, -x,		Copy the rendered snippet to the clipboard.
		Aliases: []string{"run", "exploit", "r"},
		Func: func(sc *ishell.Context) {

			fs := pflag.NewFlagSet("output", pflag.ContinueOnError)
			output := fs.StringP("output", "o", "/dev/null", "File to save output of rendering.")
			//clip := fs.BoolP("clip", "x", false, "Copy to clipboard.")
			quiet := fs.BoolP("quiet", "q", false, "Do not print to stdout.  Without --output or --clip, using this flag would make rendering useless.")
			fs.Parse(sc.Args)

			snippetText := snippetData.SnippetText

			for variable, value := range variableMap {
				snippetText = strings.ReplaceAll(snippetText, "{{ "+variable+" }}", value)
			}
			if *output != "/dev/null" {
				//fix from here
				outputDir := filepath.Dir(*output)
				sc.Println(outputDir)
				_, err := os.Stat(outputDir)
				if os.IsNotExist(err) {
					// Directory does not exist, so create it and all parent directories
					err := os.MkdirAll(outputDir, os.ModePerm)
					if err != nil {
						sc.Print("%s\n", fmt.Errorf("failed to create directory: %w", err))
						return
					}
				}
				err = os.WriteFile(*output, []byte(snippetText), 0644)
				if err != nil {
					errorMsg := fmt.Errorf("failed to create directory: %w", err)
					sc.Print("Failed to write: %s\n", errorMsg)
					return
				}

			}
			//			if *clip {
			//err := clipboard.Init()
			//if err != nil {
			//panic(err)
			//}
			//clipboard.Write(clipboard.FmtText, []byte(snippetText))

			//}
			if !*quiet {
				sc.Printf(snippetText + "\n")
			}
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
			err := browse(searchTerm, ask_db.DB, SearchField)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Ran into error executing browse function: %w", err)
			}
		},
	})

	sub.AddCmd(&ishell.Cmd{
		Name: "use",
		Help: "use an existing snippet",
		Completer: func([]string) []string {

			stmt, _ := db.Prepare("SELECT name FROM snippets")
			rows, _ := stmt.Query()

			snippetNames := []string{}
			seen := make(map[string]struct{})

			for rows.Next() {
				var snippetName string
				if err := rows.Scan(&snippetName); err != nil {
					continue
				}

				if _, exists := seen[snippetName]; !exists {
					snippetNames = append(snippetNames, snippetName)
					seen[snippetName] = struct{}{}
				}
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
			shell.Close()
			sub.Close()
		},
	})

	sub.Run()
	return nil
}
