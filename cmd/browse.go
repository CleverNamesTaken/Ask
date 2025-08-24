package cmd

import (
	"ask/ask_db"
	"ask/internal/debug"
	structs "ask/internal/model"
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func browse(searchTerm string, db *sql.DB) error {
	debug.Print("[*] Starting consoleSearch function for %s", searchTerm)

	var name, version, description, snippetID string
	matchSnippets := make(map[string]structs.Result)

	debug.Print("[*] Creating search query")

	if searchTerm == "*" {
		stmt, err := db.Prepare("SELECT id, name, version, description FROM snippets")
		debug.Print("[*] Prepared search query")
		if err != nil {
			return err
		}
		rows, err := stmt.Query()
		debug.Print("[*] Search query executed")
		if err != nil {
			debug.Print("[!] Query failed!")
			return err
		}

		for rows.Next() {
			debug.Print("[+] Iterating over the matching rows")
			if err := rows.Scan(&snippetID, &name, &version, &description); err != nil {
				log.Fatalf("Row scan failed: %v", err)
			}
			if snippetID != "" {
				tempResultStruct := structs.Result{}
				tempResultStruct.Name = name
				tempResultStruct.Version = version
				tempResultStruct.Description = description
				matchSnippets[snippetID] = tempResultStruct
			}
		}

	} else {
		stmt, err := db.Prepare("SELECT id, name, version, description FROM snippets WHERE name LIKE ?")
		if err != nil {
			return err
		}
		rows, err := stmt.Query("%" + searchTerm + "%")
		if err != nil {
			debug.Print("[!] Query failed!")
			return err
		}
		for rows.Next() {
			debug.Print("[+] Iterating over the matching rows")
			if err := rows.Scan(&snippetID, &name, &version, &description); err != nil {
				log.Fatalf("Row scan failed: %v", err)
			}
			if snippetID != "" {
				tempResultStruct := structs.Result{}
				tempResultStruct.Name = name
				tempResultStruct.Version = version
				tempResultStruct.Description = description
				matchSnippets[snippetID] = tempResultStruct
			}
		}
	}

	debug.Print("[+] Successfully created the search query")
	debug.Print("[+] Query succeeded")

	//Prepare table
	var resultsTable bytes.Buffer
	tw := table.NewWriter()
	tw.SetOutputMirror(&resultsTable)
	tw.AppendHeader(table.Row{"ID", "Name", "Version", "Description"})
	// Merge everything into headerSection

	for id, match := range matchSnippets {
		//Create the table here too
		tw.AppendRow(table.Row{id, match.Name, match.Version, match.Description})
	}
	tw.SetStyle(table.StyleRounded) // Optional: You can use StyleLight, StyleBold, etc.
	tw.Render()
	fmt.Fprintf(os.Stdout, resultsTable.String())
	return nil
}

var (
	browseCmd = &cobra.Command{
		Use:     "browse",
		Short:   "Browse snippets in the database. Aliased to 'ls' ",
		Aliases: []string{"ls"},
		Long: `Take a quick look at the snippet names, versions and descriptions for the database

EXAMPLES

ask browse
	#Quickly examine everything
ask browse example
	#Look for snippets that have "example" in the name
`,
		//Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			var searchTerm string
			if len(args) == 0 {
				searchTerm = "*"
			} else {
				// Need to fix this search string to allow any number of arguments
				searchTerm = args[0]
			}
			err := browse(searchTerm, ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Ran into error executing browse function: %w", err)
				return
			}
			fmt.Fprintf(os.Stdout, "\n\n [*] What do you want to do next?\n")
			fmt.Fprintf(os.Stdout, "\n\t ask render <SnippetName OR SnippetID>")
			fmt.Fprintf(os.Stdout, "\n\t ask cat <SnippetName OR SnippetID>")
			fmt.Fprintf(os.Stdout, "\n\t ask edit <SnippetName OR SnippetID>")
			fmt.Fprintf(os.Stdout, "\n\t ask rm <SnippetName OR SnippetID>")
			fmt.Fprintf(os.Stdout, "\n\t ask add <snippetFile OR snippetYaml OR snippetArchive>")

		},
	}
)
