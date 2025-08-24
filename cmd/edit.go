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
	debug.Print("[*] Starting to edit snippet: %s", snippetID)

	var snippetData structs.Snippet
	var yamlData structs.Snippet
	var jsonVar string
	//  query database
	stmt, err := db.Prepare("SELECT name, version, description, variables, snippetText FROM snippets WHERE id =?")

	if err != nil {
		panic(err)
	}
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

/*
func tagTest() (err error) {
	var tag string
	var tagString string

	//go doc "github.com/charmbracelet/huh".Select.OptionsFunc

	var tags []string
	form := huh.NewForm(
		//https://github.com/charmbracelet/huh/issues/500
		huh.NewGroup(
			//Need to set the position at the top for each section
			huh.NewSelect[string]().
				Title("Add tags").
				//Could see how this could be massive for the list
				OptionsFunc(func() []huh.Option[string] {
					options := make([]huh.Option[string], 3)
					options[0] = huh.NewOption("COMPLETE", "COMPLETE")
					options[1] = huh.NewOption("Custom", "Custom")

					//This should be a dynamic list based on what is in the database
					return options

				}, &tagString).
				Value(&tag).
				Validate(func(tag string) error {
					//Need to make sure we do not duplicate tags
					//should be case
					if tag == "COMPLETE" {
						return nil
					} else if tag == "Custom" {
						errorMessage := "Custom tag selection identified.  You will be prompted shortly for the tags"
						tags = append(tags, tag)
						return errors.New(errorMessage)

					}
					//if custom, handle it
					errorMessage := "Adding tag: " + tag
					tags = append(tags, tag)
					tagString = strings.Join(tags, "|")
					return errors.New(errorMessage)

				}),
			huh.NewSelect[string]().
				Title("Remove tags").
				OptionsFunc(func() []huh.Option[string] {
					removeOptions := make([]huh.Option[string], 1+len(tags))
					removeOptions[0] = huh.NewOption("None", "None")
					for i, tag := range tags {

						removeOptions[i+1] = huh.NewOption(tag, tag)
					}
					return removeOptions

				}, &tagString).
				Value(&tag).
				Validate(func(tag string) error {
					if tag == "None" {
						return nil
					} else {
						errorMessage := "Removing tag: " + tag
						for i := range tags {
							if tags[i] == tag {
								tags = append(tags[:i], tags[i+1:]...)
								break
							}
						}
						tagString = strings.Join(tags, "|")

						return errors.New(errorMessage)
					}
				}),
		).WithHeight(10),
	)

	err = form.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(strings.Join(tags, "|"))

	return nil
}
*/

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
