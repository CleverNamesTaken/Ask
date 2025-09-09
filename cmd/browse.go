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
	"slices"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// SearchField is a flag variable
var SearchField []string

// SearchAll is a flag variable
var SearchAll bool

func browse(searchTerm string, db *sql.DB, SearchField []string) error {
	debug.Print("[*] Entered browse function.")

	// validate search fields
	if len(SearchField) == 0 {
		SearchField = append(SearchField, "name")
	} else {
		for _, field := range SearchField {
			if !slices.Contains([]string{"name", "tag", "desc", "raw", "tags"}, field) {

				return fmt.Errorf("[!] Invalid search field found: %s\n[*]Valid search fields are: name, tag, desc, raw", field)
			}
		}
	}

	var name, version, description, snippetID string
	var idSlice []string
	//check search term
	//if *, then grab everything

	if searchTerm == "*" {
		rows, err := db.Query("SELECT id FROM snippets")
		if err != nil {
			return err
		}
		for rows.Next() {
			if err := rows.Scan(&snippetID); err != nil {
				log.Fatalf("Row scan failed: %v", err)
			}
			idSlice = append(idSlice, snippetID)
		}
	} else {
		//otherwise, check searchfield and create map for slices
		//these searches are case sensitive right now

		for _, field := range SearchField {
			switch field {
			case "name":

				rows, err := db.Query("SELECT id FROM snippets WHERE name like ?", "%"+searchTerm+"%")
				if err != nil {
					return err
				}
				for rows.Next() {
					if err := rows.Scan(&snippetID); err != nil {
						log.Fatalf("Row scan failed: %v", err)
					}
					idSlice = append(idSlice, snippetID)
				}

				//probably a better way to simpify this rather than having tag and tags
			case "tag":

				var tagID string
				db.QueryRow("SELECT id FROM tags WHERE tag LIKE ?", "%"+searchTerm+"%").Scan(&tagID)
				rows, err := db.Query("SELECT snipId FROM tagMap WHERE tagId = ?", tagID)
				if err != nil {
					return err
				}
				for rows.Next() {
					if err := rows.Scan(&snippetID); err != nil {
						log.Fatalf("Row scan failed: %v", err)
					}
					idSlice = append(idSlice, snippetID)
				}

			case "tags":

				var tagID string
				db.QueryRow("SELECT id FROM tags WHERE tag LIKE ?", "%"+searchTerm+"%").Scan(&tagID)
				rows, err := db.Query("SELECT snipId FROM tagMap WHERE tagId = ?", tagID)
				if err != nil {
					return err
				}
				for rows.Next() {
					if err := rows.Scan(&snippetID); err != nil {
						log.Fatalf("Row scan failed: %v", err)
					}
					idSlice = append(idSlice, snippetID)
				}

			case "desc":

				rows, err := db.Query("SELECT id FROM snippets WHERE description LIKE ?", "%"+searchTerm+"%")
				if err != nil {
					return err
				}
				for rows.Next() {
					if err := rows.Scan(&snippetID); err != nil {
						log.Fatalf("Row scan failed: %v", err)
					}
					idSlice = append(idSlice, snippetID)
				}

			case "raw":

				rows, err := db.Query("SELECT id FROM snippets WHERE snippetText LIKE ?", "%"+searchTerm+"%")
				if err != nil {
					return err
				}
				for rows.Next() {
					if err := rows.Scan(&snippetID); err != nil {
						log.Fatalf("Row scan failed: %v", err)
					}
					idSlice = append(idSlice, snippetID)
				}

			}

		}
	}

	//Remove any snippets that are not the most recent version
	if !SearchAll {
		var latestVersionSlice []string
		for _, id := range idSlice {
			var name string
			var version string
			err := db.QueryRow("SELECT name,version FROM snippets WHERE id = ?", id).Scan(&name, &version)
			if err != nil {
				return err
			}
			latestVersionStruct, _ := getLatestVersion(name, db)
			latestVersion := fmt.Sprintf("%d.%d.%d", latestVersionStruct.Major, latestVersionStruct.Minor, latestVersionStruct.Patch)
			if version == latestVersion {
				latestVersionSlice = append(latestVersionSlice, id)
			}

		}
		idSlice = latestVersionSlice
	}

	//loop through slice to get results
	matchSnippets := make(map[string]structs.Result)
	for _, snippetID = range idSlice {
		err := db.QueryRow("SELECT name,version,description FROM snippets WHERE id = ?", snippetID).Scan(&name, &version, &description)
		if err != nil {
			return err
		}
		tempResultStruct := structs.Result{}
		tempResultStruct.Name = name
		tempResultStruct.Version = version
		tempResultStruct.Description = description
		matchSnippets[snippetID] = tempResultStruct
	}

	//Prepare table
	var resultsTable bytes.Buffer
	tw := table.NewWriter()
	tw.SetOutputMirror(&resultsTable)

	if !SearchAll {
		tw.AppendHeader(table.Row{"ID", "Name", "Description", "Tags"})
	} else {
		tw.AppendHeader(table.Row{"ID", "Name", "Version", "Description", "Tags"})
	}
	// Merge everything into headerSection

	for id, match := range matchSnippets {

		//grab all the tags based on the id

		existingTagMap, err := getTags(id, db)
		if err != nil {
			log.Fatal(err)
		}
		var Tags []string
		for tag, _ := range existingTagMap {
			Tags = append(Tags, tag)
		}
		tagString := strings.Join(Tags, ", ")

		//Create the table here too

		if !SearchAll {
			tw.AppendRow(table.Row{id, match.Name, match.Description, tagString})
		} else {
			tw.AppendRow(table.Row{id, match.Name, match.Version, match.Description, tagString})
		}
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
		Long: `Take a quick look at the snippet names, versions, tags and descriptions for the database.  The --field flag can be used to designate which field to search among the choices "name","tag" ,"raw" or "desc".  Searches are case sensitive and will search the snippet name, tags, raw snippet text and snippet description, respectively.

EXAMPLES

ask browse
	#Quickly examine everything
ask browse example --all
	#Look for snippets that have "example" in the name and show all versions instead of just the most recent version
ask ls -f tag web
	#Look for snippets that have a web tag
ask ls -f tag -f raw ssh
	#Look for snippets that have ssh in the tag name or in the raw snippet text.
`,
		//Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			var searchTerm string
			if len(args) == 0 {
				searchTerm = "*"
			} else {
				// Need to fix this search string to allow any number of arguments
				searchTerm = strings.Join(args, " ")
			}
			err := browse(searchTerm, ask_db.DB, SearchField)
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

func init() {
	browseCmd.PersistentFlags().StringSliceVarP(&SearchField, "field", "f", []string{}, "Which field(s) among name, tag, description, and raw for a case-sensitive search.  Use multiple flags to search multiple fields. (Default selection is name.)")
	browseCmd.PersistentFlags().BoolVarP(&SearchAll, "all", "a", false, "Show the version number and all results instead of just the most recent snippet version.")
}
