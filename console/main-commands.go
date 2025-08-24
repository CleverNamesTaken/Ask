package cmd

import (
	"fmt"
	"os"
	"go-ask/internal/debug"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"bytes"
	"log"
	"go-ask/ask_db"
	"github.com/jedib0t/go-pretty/v6/table"
	structs "go-ask/internal/model"
	//"strconv"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

func subshell(snippetId string) error {
	//Enter a subshell to explore a snippet, set variables and render a snippet
	debug.Print("[*] Entering subshell for snippetID: %s", snippetId)
	fmt.Printf("[*] Entering subshell for snippetID: %s\n", snippetId)


	//show info
	//show options
	//set
	//setg
	//render, aliases of use, exploit
	//back
	//still allow search and use functions

	return nil
}

func consoleSearch(searchTerm string, db *sql.DB) error {
	debug.Print("[*] Starting consoleSearch function for %s", searchTerm)

	var name, version, description, snippetId string
	matchSnippets := make(map[string]structs.Result)
	

	debug.Print("[*] Creating search query")

	if searchTerm == "*" {
		stmt, err := db.Prepare("SELECT id, name, version, description FROM snippets")
		if err != nil {
			return err
		}
		rows, err := stmt.Query()
		if err != nil {
			debug.Print("[!] Query failed!")
			return err
		}

		for rows.Next() {
			debug.Print("[+] Iterating over the matching rows")
			if err := rows.Scan(&snippetId, &name, &version, &description); err != nil {
				log.Fatalf("Row scan failed: %v", err)
			}
			if snippetId != "" {
				tempResultStruct := structs.Result{}
				tempResultStruct.Name = name
				tempResultStruct.Version = version
				tempResultStruct.Description = description
				matchSnippets[snippetId] = tempResultStruct
			}
		}




	} else {
		stmt, err := db.Prepare("SELECT id, name, version, description FROM snippets WHERE name LIKE ?")
		if err != nil {
			return err
		}
		rows, err := stmt.Query("%"+searchTerm+"%")
		if err != nil {
			debug.Print("[!] Query failed!")
			return err
		}
		for rows.Next() {
			debug.Print("[+] Iterating over the matching rows")
			if err := rows.Scan(&snippetId, &name, &version, &description); err != nil {
				log.Fatalf("Row scan failed: %v", err)
			}
			if snippetId != "" {
				tempResultStruct := structs.Result{}
				tempResultStruct.Name = name
				tempResultStruct.Version = version
				tempResultStruct.Description = description
				matchSnippets[snippetId] = tempResultStruct
			}
		}
	}

	debug.Print("[+] Successfully created the search query")
	debug.Print("[+] Query succeeded")


	//Prepare table
	var resultsTable bytes.Buffer
	tw := table.NewWriter()
	tw.SetOutputMirror(&resultsTable)
	tw.AppendHeader(table.Row{"ID", "Name","Version", "Description"})
	// Merge everything into headerSection


	for id, match := range matchSnippets {
		//Create the table here too
		tw.AppendRow(table.Row{id,match.Name,match.Version,match.Description})
	}
	tw.SetStyle(table.StyleRounded) // Optional: You can use StyleLight, StyleBold, etc.
	tw.Render()
	fmt.Fprintf(os.Stdout,resultsTable.String())
	fmt.Fprintf(os.Stdout,"\n\n [+] Next you can use ")
	return nil
}


func consoleUse(snippetString string, db *sql.DB) (err error) {
	debug.Print("Started consoleUse function with snippetString of %s\n",snippetString)

	//Determine if it is an id or a snippet name

	snippetId, _, err := getSnippetId(snippetString, db)
	if err != nil {
		fmt.Fprintf(os.Stdout,"[!] Snippet associated with %s not found.  Please use the search function to find a valid id or snippet name.",snippetString)
		return err
	}
	//Drop into a sub shell
	err = subshell(snippetId)
	if err != nil {
		return err
	}

	//Incorporate set, setg, info, show options, show info, whatever.

	return nil
}
func mainMenuCommands(app *console.Console) console.Commands {
	return func() *cobra.Command {
		rootCmd := &cobra.Command{}
		rootCmd.Short = shortUsage

		exitCmd := &cobra.Command{
			Use:   "exit",
			Short: "Exit the console application",
			Run: func(cmd *cobra.Command, args []string) {
				exitCtrlD(app)
			},
		}
		rootCmd.AddCommand(exitCmd)

		useCmd := &cobra.Command{
			Use:   "use",
			Short: "Select a snippet to use",
			Long: `Select a snippet to use either with the snippet number, the snippet name, or tab completion.  Use the search function to obtain snippet IDs or names.`,
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				//Should create a completions function for snippet based on something similar to how metasploit works 
				consoleUse(args[0])
				fmt.Println("Use a snippet")
			},
		}
		rootCmd.AddCommand(useCmd)

		searchCmd := &cobra.Command{
			Use:   "search",
			Short: "Search for a snippet",
			Run: func(cmd *cobra.Command, args []string) {
				var searchTerm string
				if len(args) == 0 {
					searchTerm = "*"
				} else {
					// Need to fix this search string to allow any number of arguments
					searchTerm = args[0]
				}
				err := consoleSearch(searchTerm, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stdout,"[!] Ran into error executing consoleSearch function: %w",err)
					return
				}
			},
		}
		rootCmd.AddCommand(searchCmd)

		// Completions ----------------------------------------------------------------- //
		//

		// For each of the commands above, generate the carapace.Carapace for the command.
		// Then create a map carapace.FlagMap, and add file completion to all flags requiring
		// a file argument.
		/*
			for _, cmd := range rootCmd.Commands() {
				c := carapace.Gen(cmd)

				if cmd.Args != nil {
					c.PositionalAnyCompletion(
						carapace.ActionCallback(func(c carapace.Context) carapace.Action {
							return carapace.ActionFiles()
						}),
					)
				}

				flagMap := make(carapace.ActionMap)

				cmd.Flags().VisitAll(func(f *pflag.Flag) {
					if f.Name == "file" || strings.Contains(f.Usage, "file") {
						flagMap[f.Name] = carapace.ActionFiles()
					}
				})

				if cmd.Name() == "ssh" {
					// Generate a list of random hosts to use as positional arguments
					hosts := make([]string, 0)
					for i := 0; i < 10; i++ {
						hosts = append(hosts, fmt.Sprintf("host%d", i))
					}

					c.PositionalCompletion(carapace.ActionValues(hosts...))
				}

				if cmd.Name() == "encrypt" {
					cmd.Flags().VisitAll(func(f *pflag.Flag) {
						if f.Name == "algorithm" {
							flagMap[f.Name] = carapace.ActionValues("aes", "des", "blowfish")
						}
					})
				}

				c.FlagCompletion(flagMap)
			}
		*/

		rootCmd.InitDefaultHelpCmd()
		rootCmd.CompletionOptions.DisableDefaultCmd = true
		rootCmd.DisableFlagsInUseLine = true

		return rootCmd
	}
}
