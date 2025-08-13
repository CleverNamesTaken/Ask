package cmd

import (
	//"errors"
	"fmt"
	"os"

	//"strconv"
	//"strings"
	//"time"
	"database/sql"
	"log"

	"ask/internal/debug"

	"github.com/charmbracelet/huh"
	//"github.com/charmbracelet/huh/spinner"
	//"github.com/charmbracelet/lipgloss"
	//xstrings "github.com/charmbracelet/x/exp/strings"
)

// List all the available snippets
func listSnippets(db *sql.DB) (snippetID, snippetName string) {
	debug.Print("[*] Entered listSnippets function")
	var name, version, description string
	allSnippets := make(map[string]string)

	rows, err := db.Query("SELECT id, name, version, description FROM snippets")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for rows.Next() {
		if err := rows.Scan(&snippetID, &name, &version, &description); err != nil {
			log.Fatalf("Row scan failed: %v", err)
		}
		if snippetID != "" {
			allSnippets[snippetID] = fmt.Sprintf("%-30s v%-6s %-50s", name, version, description)
		}
	}

	var options []huh.Option[string]
	for id, display := range allSnippets {
		newOptions := huh.NewOption(display, id)
		options = append(options, newOptions)
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Snippet").
				Description("This is a description").
				Options(
					options...,
				).
				Value(&snippetID),
		),
	)

	err = form.Run()
	if err != nil {
		fmt.Println("Uh oh:", err)
		os.Exit(1)
	}

	query := `SELECT name FROM snippets WHERE id = ?`
	row := db.QueryRow(query, snippetID)

	row.Scan(&name)

	return snippetID, name
}

func init() {
}
