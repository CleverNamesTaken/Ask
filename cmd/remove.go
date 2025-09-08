package cmd

import (
	"ask/ask_db"
	"ask/internal/debug"
	structs "ask/internal/model"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// ForceRemove is a flag variable
var ForceRemove bool

// Prune is a flag variable
var Prune bool

func checkSnippetIDExists(snippetID string, db *sql.DB) (exists bool) {
	// check the database to see if the snippetID is valid
	debug.Print("[*] Checking if snippet exists with id %s\n", snippetID)
	query := `SELECT EXISTS(SELECT 1 FROM snippets WHERE id = ?)`
	err := db.QueryRow(query, snippetID).Scan(&exists)
	if err != nil {
		// Log or handle the error based on your preference
		log.Printf("[!] Error checking snippetID: %v", err)
		return false
	}
	debug.Print("[*] id exists %s\n", exists)
	return exists
}

func getSnippetID(snippetString string, db *sql.DB) (snippetID string, snippetName string, err error) {
	debug.Print("[*] Getting snippet ID: %s", snippetString)
	// Try to parse as integer; if successful, assume it's an ID and return it
	if _, parseErr := strconv.Atoi(snippetString); parseErr == nil {
		debug.Print("[*] Supplied argument appears to be a snippetID: %s", snippetString)
		query := `SELECT name FROM snippets WHERE id = ?`
		row := db.QueryRow(query, snippetString)

		err = row.Scan(&snippetName)
		if err != nil {
			return snippetID, "", fmt.Errorf("Snippet not found: %s\nPerhaps try\nask ls", snippetString)
		}
		return snippetString, snippetName, nil
	}

	// Otherwise, treat as name and look up the latest version's ID
	query := `SELECT id FROM snippets WHERE name = ? ORDER BY id DESC LIMIT 1`
	row := db.QueryRow(query, snippetString)

	// Scan the result into snippetID
	err = row.Scan(&snippetID)
	if err != nil {
		return "", "", fmt.Errorf("Snippet not found: %s\nPerhaps try\nask ls", snippetString)
	}
	debug.Print("[*] Identified snippet ID: %s", snippetID)

	return snippetID, snippetString, nil
}

func isLatestSnippet(db *sql.DB, snippetID string, snippetName string) bool {
	debug.Print("[*] Checking if snippet is the latest version: %s", snippetID)
	var version string
	var latestVersion string
	var latestVersionStruct structs.Version

	query := `SELECT version FROM snippets WHERE id = ?`
	row := db.QueryRow(query, snippetID)

	debug.Print("[*] Querying current version : %s", query)
	if err := row.Scan(&version); err != nil {
		fmt.Fprintf(os.Stderr, "[!] Ran into an error looking up the version number for %s, id %s\n", snippetName, snippetID)
		fmt.Println("%w", err)
	}
	debug.Print("[*] Found version number: %s", version)
	latestVersionStruct, err := getLatestVersion(snippetName, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Ran into an error looking up the latest version number for %s\n", snippetName)
	}

	latestVersion = fmt.Sprintf("%d.%d.%d", latestVersionStruct.Major, latestVersionStruct.Minor, latestVersionStruct.Patch)
	debug.Print("[*] Comparing latest version to this version: %s vs %s", latestVersion, version)
	if latestVersion == version {
		debug.Print("[*] Snippet is latest version: %s", snippetID)
		return true

	}
	debug.Print("[*] Snippet is not latest version: %s", snippetID)
	return false

}

func getVersion(snippetID string, db *sql.DB) (snippetVersion string, err error) {
	//Given a snippet ID, get the associated version number
	query := `SELECT version FROM snippets WHERE id = ?`
	row := db.QueryRow(query, snippetID)
	err = row.Scan(&snippetVersion)
	return snippetVersion, err
}

func getTags(snippetID string, db *sql.DB) (tagMap map[string]string, err error) {
	//Given a snippet ID, get the associated tag name and ids
	tagMap = make(map[string]string)

	query := `SELECT tagId FROM tagMap WHERE snipId = ?`
	rows, err := db.Query(query, snippetID)

	if err != nil {
		return tagMap, err
	}

	var tagId string
	var tag string

	for rows.Next() {
		if err := rows.Scan(&tagId); err != nil {
			return tagMap, err
		}
		db.QueryRow("SELECT tag FROM tags WHERE id = ?", tagId).Scan(&tag)
		debug.Print("[+] Found tag: %s", tag)
		tagMap[tag] = tagId
	}

	return tagMap, nil
}

func remove(snippetID string, snippetName string, db *sql.DB) (err error) {
	debug.Print("[*] Trying to remove %s , number %s\n", snippetName, snippetID)

	if isLatestSnippet(db, snippetID, snippetName) && !ForceRemove {
		fmt.Print("[!] This is the latest snippet version. Are you sure you want to remove it? (y/yes to confirm): ")
		var prompt string
		if _, err := fmt.Scanln(&prompt); err != nil || (strings.ToLower(prompt) != "y" && strings.ToLower(prompt) != "yes") {
			fmt.Fprintf(os.Stdout, "[!] Removal cancelled")
			return nil
		}
	}

	snippetVersion, err := getVersion(snippetID, db)
	if err != nil {
		return err
	}

	snippetTags, err := getTags(snippetID, db)

	_, err = db.Exec("DELETE FROM snippets WHERE id = ?", snippetID)
	if err != nil {
		fmt.Fprintf(os.Stdout, "[!] Failed to remove snippet.\n")
		return err
	}

	_, err = db.Exec("DELETE FROM tagMap WHERE snipId = ?", snippetID)
	if err != nil {
		fmt.Fprintf(os.Stdout, "[!] Failed to remove tags for snippet.\n")
		return err
	}

	fmt.Fprintf(os.Stdout, "[+] Snippet removed: %s_v%s\n", snippetName, snippetVersion)

	//check if any other snippets have the tags removed.  If not, then remove the entry from the tags table
	var exists bool
	for _, tagId := range snippetTags {
		query := `SELECT EXISTS(SELECT 1 FROM tagMap WHERE tagId = ?)`
		db.QueryRow(query, tagId).Scan(&exists)
		if !exists {
			debug.Print("[!] Removed tagId: %s", tagId)
			_, err = db.Exec("DELETE FROM tags WHERE id = ?", tagId)
			if err != nil {
				return err
			}

		}

	}
	return nil
}

var (
	removeCmd = &cobra.Command{
		Use:     "remove",
		Short:   "Remove snippets from the database. Aliased to 'rm' ",
		Aliases: []string{"rm"},
		Long: `Remove snippet from the database.  Note that you will be prompted to confirm removal of any snippet that is the most updated version of itself, but not anything that is outdated.  Removal is non-reversible.

EXAMPLES

ask remove
	#Identify which snippet to remove, and then do so
ask remove example
	#Remove the latest version of example.  The older versions will remain
ask remove 1337 --force
	#Remove the snippet with id 1337, and do it without being prompted even if it is the latest version
ask remove --prune
	#Remove all snippets that are not the latest version
`,
		//Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if Prune == true {
				prune(ask_db.DB)
				return
			}

			if len(args) == 0 {
				snippetID, snippetName := listSnippets(ask_db.DB)
				remove(snippetID, snippetName, ask_db.DB)
			} else {
				snippetString := args[0]
				//get snippet id or name
				snippetID, snippetName, err := getSnippetID(snippetString, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stdout, "[!] Error: %s\n", err)
					return
				}
				if checkSnippetIDExists(snippetID, ask_db.DB) == true {
					remove(snippetID, snippetName, ask_db.DB)
				} else {
					fmt.Fprintf(os.Stdout, "[!] Snippet name or id does not exist in database: %s\n", snippetString)
				}
			}
		},
	}
)

func prune(db *sql.DB) {
	query := `
    SELECT s.id
    FROM snippets s
    JOIN (
        SELECT name, MAX(version) AS max_version
        FROM snippets
        GROUP BY name
    ) mx ON s.name = mx.name
    WHERE s.version < mx.max_version
    `
	rows, err := db.Query(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] error fetching preserve IDs: %w\n", err)
		return
	}
	defer rows.Close()

	var toRemove []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return
		}
		toRemove = append(toRemove, id)
	}

	if len(toRemove) == 0 {
		fmt.Fprintf(os.Stdout, "[+] Database is all pruned up -- nothing to remove\n")
		return
	}

	for _, id := range toRemove {
		snippetID, snippetName, err := getSnippetID(strconv.Itoa(id), db)
		if err != nil {
			log.Printf("[!] Failed to get snippet id: %w", err)
			return
		}
		if err := remove(snippetID, snippetName, db); err != nil {
			log.Printf("[!] failed to remove snippet ID %d: %v\n", id, err)
		}
	}

	fmt.Fprintf(os.Stdout, "[+] Pruned old versions\n")
}

func init() {
	removeCmd.Flags().BoolVarP(&ForceRemove, "force", "f", false, "Do not prompt for removal of a snippet that is the latest version.")
	removeCmd.Flags().BoolVarP(&Prune, "prune", "p", false, "Remove all snippets that are out of date.")

}
