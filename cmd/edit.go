// Package cmd contains all the cobra commands
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"

	"ask/ask_db"
	"io/ioutil"

	"github.com/spf13/cobra"

	"database/sql"

	//We might need this driver
	_ "github.com/go-sql-driver/mysql"

	"ask/internal/debug"
	structs "ask/internal/model"

	"log"
	"os"

	yaml "gopkg.in/yaml.v3"
)

func edit(snippetID string, db *sql.DB) {
	//edit a snippet based on a snippet id
	debug.Print("[*] Starting to edit snippet id #: %s", snippetID)

	var snippetData structs.Snippet
	var yamlData structs.Snippet
	var jsonVar string
	//  query database
	stmt, err := db.Prepare("SELECT name, version, description, variables, snippetText FROM snippets WHERE id =?")

	if err != nil {
		panic(err)
	}
	//should use query row because the id is unique
	rows, err := stmt.Query(snippetID)
	if err != nil {
		panic(err)
	}
	// make sure the snippetID exists

	for rows.Next() {
		err := rows.Scan(&snippetData.Name, &snippetData.Version, &snippetData.Description, &jsonVar, &snippetData.SnippetText)
		if err != nil {
			log.Fatal(err)
		}
	}
	// convert json string to Variables struct

	json.Unmarshal([]byte(jsonVar), &snippetData.Variables)

	//  Prepare variables for form
	SnippetText := snippetData.SnippetText
	SnippetVersion := snippetData.Version
	// check if we are trying to update an older version and ask if we really want to do that []
	snippetTextHandle, err := ioutil.TempFile("", "temp-*.txt")
	snippetFile := snippetTextHandle.Name()
	snippetTextHandle.Close()

	//prepare tags for the form too
	existingTagMap, err := getTags(snippetID, db)
	if err != nil {
		log.Fatal(err)
	}
	for tag, _ := range existingTagMap {
		snippetData.Tags = append(snippetData.Tags, tag)
	}

	yamlByteString, err := yaml.Marshal(snippetData)
	yamlString := string(yamlByteString)

	// Open up text editors
	SnippetText, yamlString = editForm(snippetData, snippetFile, SnippetVersion, db)

	yaml.Unmarshal([]byte(yamlString), &snippetData)

	//Add back the snippetFile and text
	snippetData.SnippetFile = snippetFile
	snippetData.SnippetText = SnippetText

	//  Create temp file
	tempYamlHandle, err := ioutil.TempFile("", "temp-*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	yamlFile := tempYamlHandle.Name()
	tempYamlHandle.Close()

	snippetYamlHandle := writeYaml(snippetData, yamlFile)
	snippetYamlHandle.Close()

	if err != nil {
		log.Fatalf("Unable to write snippet temp file: %s", err)
	}

	os.WriteFile(snippetFile, []byte(snippetData.SnippetText), 0644)

	//  lint

	lintRequired := true
	for {
		if lintRequired == true {
			// need to pass more for editForm
			lintRequired = lintPass(SnippetText, yamlString, snippetFile, yamlFile, db)
			//  grab new SnippetText and yamlString
			yamlData, _ = ReadYAML[structs.Snippet](yamlFile)

			yamlByteString, err = yaml.Marshal(yamlData)

			SnippetText = yamlData.SnippetText
			yamlString = string(yamlByteString)
		} else {
			break
		}
	}

	//  keep going until done or we give up
	updateDatabase(yamlData, db)

	os.Remove(snippetFile)
	os.Remove(yamlFile)
	fmt.Fprintf(os.Stdout, "[+] Edited snippet: %s_v%s", yamlData.Name, yamlData.Version)
	return
}

func editForm(snippetStruct structs.Snippet, snippetFile string, newVersion string, db *sql.DB) (SnippetText string, yamlString string) {
	//run a form to help edit
	var updateType string

	textGroup := huh.NewGroup(
		huh.NewText().
			Editor("vim").
			Lines(30).
			Title("Snippet text").
			Value(&snippetStruct.SnippetText).
			WithHeight(25),
	)
	form := huh.NewForm(
		textGroup,
	)
	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	// I need to compare the variables found with what is already in the database
	variables, err := findVariables(snippetStruct.SnippetText)

	for variable := range variables {
		if _, exists := snippetStruct.Variables[variable]; !exists {
			snippetStruct.Variables[variable] = &structs.Variable{} // zero-value struct
		}
	}

	// Remove keys from `snippetStruct.Variables` that aren't in `keys`
	for key := range snippetStruct.Variables {
		if _, keep := variables[key]; !keep {
			delete(snippetStruct.Variables, key)
		}
	}

	snippetStruct = snippetWizard(snippetStruct, db)

	form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				TitleFunc(func() string {
					latestVersionStruct, _ := getLatestVersion(snippetStruct.Name, db)
					latestVersion := fmt.Sprintf("%d.%d.%d", latestVersionStruct.Major, latestVersionStruct.Minor, latestVersionStruct.Patch)
					title := fmt.Sprintf("Patch type. Current version is %s", latestVersion)
					return title
				}, &updateType).
				Options(
					huh.NewOption("Patch", "Patch"),
					huh.NewOption("Minor", "Minor"),
					huh.NewOption("Major", "Major"),
					huh.NewOption("Custom", "Custom"),
				).
				Value(&updateType).
				Validate(func(updateType string) error {
					if updateType == "Custom" {
						return nil
					}
					newVersion, _ = versionBump(snippetStruct.Version, updateType)
					latestVersion, _ := getLatestVersion(snippetStruct.Name, db)
					updatePossible, _ := updatingVersion(newVersion, latestVersion)
					if updatePossible {
						return nil
					}
					errorMessage := fmt.Sprintf("[!] Proposed version is %s, too low", newVersion)
					return errors.New(errorMessage)
				}),
		),
	)

	err = form.Run()
	if err != nil {
		log.Fatal(err)
	}

	if updateType == "Custom" {
		newVersion, err = customVersionForm(snippetStruct.Name, db)
		if err != nil {
			log.Fatalf("Unable to provide custom version: %s", err)
		}
	}

	os.WriteFile(snippetFile, []byte(snippetStruct.SnippetText), 0644)

	//do logic to bump up the version number
	debug.Print("[*] Proposed version is %s", newVersion)
	snippetStruct.Version = newVersion

	yamlByteString, err := yaml.Marshal(snippetStruct)
	snippetYaml := string(yamlByteString)
	return snippetStruct.SnippetText, snippetYaml

}
func customVersionForm(snippetName string, db *sql.DB) (newVersion string, err error) {
	//run form for custom versioning
	latestVersionStruct, _ := getLatestVersion(snippetName, db)
	latestVersion := fmt.Sprintf("%d.%d.%d", latestVersionStruct.Major, latestVersionStruct.Minor, latestVersionStruct.Patch)
	newVersion = latestVersion

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Enter custom version number").
				Value(&newVersion).
				Validate(func(_ string) error {
					updatePossible, _ := updatingVersion(newVersion, latestVersionStruct)
					if updatePossible {
						return nil
					}
					errorMessage := fmt.Sprintf("[!] Proposed version is %s, too low.  Must be greater than %s", newVersion, latestVersion)
					return errors.New(errorMessage)
				}),
		),
	)
	err = form.Run()
	if err != nil {
		log.Fatal(err)
	}
	debug.Print("[*] Custom version : %s", newVersion)
	return newVersion, nil
}

var (
	editCmd = &cobra.Command{
		Use:   "edit",
		Short: "Edit an existing snippet",
		Long: `If you know the name of the snippet, you can provide it as an argument. If you are not sure, then do not provide an argument and you can select it from the TUI menu.

EXAMPLES
ask edit
	#Take a look at all the snippets and select which one to edit

ask edit example
	#Edit the latest version of the snippet named example

ask edit 1337
	#Edit snippet number 1337

Note that the TUI will prompt you to determine what kind of update you are making -- patch, minor, major or custom.  You cannot update a snippet if its version will be less than the most-up-to-date version.  Select the "custom" update option if you are trying to leapfrog an existing snippet.`,
		Args: cobra.MinimumNArgs(0),
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 0 {
				snippetID, _ := listSnippets(ask_db.DB)
				edit(snippetID, ask_db.DB)
			} else {

				snippetString := args[0]
				//get snippet id or name
				snippetID, snippetString, err := getSnippetID(snippetString, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stdout, "[!] Error: %w\n", err)
					return
				}
				if checkSnippetIDExists(snippetID, ask_db.DB) == true {
					edit(snippetID, ask_db.DB)
				} else {
					fmt.Fprintf(os.Stdout, "[!] Snippet name or id does not exist in database: %s\n", snippetString)
				}

			}
		},
	}
)
