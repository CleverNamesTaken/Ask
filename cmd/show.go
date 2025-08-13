package cmd

import (
	"fmt"
	"ask/ask_db"
	"os"

	"database/sql"
	"ask/internal/debug"

	"github.com/spf13/cobra"
)

var (
	showCmd = &cobra.Command{
		Use:   "show",
		Short: "Take a quick peek at a raw snippet, excluding the metadata.  Aliased to 'cat'",
		Long: `The show command is helpful for quickly examining a snippet without any of the metadata.  
If you are not sure which snippet you want to look at, run the command without any arguments

EXAMPLES
ask show
	#Look through the snippets to see which one you'd like to examine
ask show example
	#Examine what the latest version of the snippet called "example" looks like
ask show 1337
	#Examine what the snippet with the snippet id of 1337 looks like
		`,
		Args:    cobra.MinimumNArgs(0),
		Aliases: []string{"cat"},
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 0 {
				snippetID, _ := listSnippets(ask_db.DB)
				show(snippetID, ask_db.DB)
			} else {

				snippetString := args[0]
				//get snippet id or name
				snippetID, snippetString, err := getSnippetID(snippetString, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stdout, "[!] Error: %s\n", err)
					return
				}
				if checkSnippetIDExists(snippetID, ask_db.DB) == true {
					show(snippetID, ask_db.DB)
				} else {
					fmt.Fprintf(os.Stdout, "[!] Snippet name or id does not exist in database: %s\n", snippetString)
				}

			}

		},
	}
)

func show(snippetID string, db *sql.DB) error {
	//Dump the snippetText to standard out
	debug.Print("[*] Dumping the text for snippet: %s", snippetID)

	var snippetText string
	//  query database
	stmt, err := db.Prepare("SELECT snippetText FROM snippets WHERE id =?")

	if err != nil {
		panic(err)
	}
	rows, err := stmt.Query(snippetID)
	if err != nil {
		panic(err)
	}
	// make sure the snippetID exists

	for rows.Next() {
		err := rows.Scan(&snippetText)
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stdout, "\n\n\n-------RAW SNIPPET------\n%s\n-------SNIPPET END------\n\n", snippetText)
	return nil
}
