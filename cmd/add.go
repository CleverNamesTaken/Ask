// Package cmd contains all the cobra commands
package cmd

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"ask/ask_db"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"ask/internal/debug"
	structs "ask/internal/model"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	//_ "github.com/go-sql-driver/mysql"
	yaml "gopkg.in/yaml.v3"
)

func quickSnippet(db *sql.DB) error {
	//Create a form to add a snippet and then ingest it
	debug.Print("Added new snippet")
	snippetStruct := structs.Snippet{}
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
		return err
	}

	// I need to compare the variables found with what is already in the database
	variables, err := findVariables(snippetStruct.SnippetText)

	debug.Print("Adding variables to struct")

	Variables := make(map[string]*structs.Variable, len(variables))
	for key := range variables {
		Variables[key] = &structs.Variable{}
	}
	// add it to the new struct
	snippetStruct.Variables = Variables
	snippetStruct = snippetWizard(snippetStruct, db)

	//this is copied code -- clean it up

	// write the yaml

	tempHandle, err := ioutil.TempFile("", "temp-*.txt")
	if err != nil {
		return err
	}

	defer tempHandle.Close()
	snippetFile := tempHandle.Name()

	os.WriteFile(snippetFile, []byte(snippetStruct.SnippetText), 0644)
	snippetStruct.SnippetFile = snippetFile
	snippetStruct.Version = "0.1.0"

	//  Create temp file
	tempYamlHandle, err := ioutil.TempFile("", "temp-*.yaml")
	if err != nil {
		return err
	}

	yamlHandle := writeYaml(snippetStruct, tempYamlHandle.Name())
	yamlFile := yamlHandle.Name()

	yamlData, _ := ReadYAML[structs.Snippet](yamlFile)
	yamlNew, _ := yaml.Marshal(yamlData)
	yamlString := string(yamlNew)

	// Linter
	if lintPass(snippetStruct.SnippetText, yamlString, snippetFile, yamlFile, db) != false {
		log.Printf("[!] Lint failed.\n")
	}

	// Update
	updateDatabase(yamlData, db)

	// Close up the database and the temp file
	yamlHandle.Close()
	os.Remove(yamlHandle.Name())
	fmt.Fprintf(os.Stdout, "[+] Added new snippet: %s_v%s", yamlData.Name, yamlData.Version)
	return nil
}

var (
	addCmd = &cobra.Command{
		Use:   "add",
		Short: "Add new snippets to the database",
		Long: `Add snippets from text, YAML, or ZIP files. If no file is provided, 
the command will prompt the user to input the snippet.

Examples:
ask add
	#Start blank snippet
ask add example.txt
	#Add the contents of example.txt as a snippet
ask add example.yaml
	#Add the contents and metadata in example.yaml
ask add archived.zip
	#Add the saved ask archive in achive.zip`,

		//Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 0 {
				err := quickSnippet(ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stdout, "[!] Encountered an error creating quick snippet: %w\n", err)
				}
				return
			}
			snippetFile := args[0]
			ext := filepath.Ext(snippetFile)
			if strings.HasSuffix(ext, "yaml") || strings.HasSuffix(ext, "yml") {
				err := ingestYaml(snippetFile, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stdout, "[!] Encountered an error ingesting yaml file: %w\n", err)
				}
				return
			}
			if strings.HasSuffix(ext, "zip") {
				//do this
				err := ingestZip(snippetFile, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stdout, "[!] Encountered an error ingesting zip file: %w\n", err)
				}
				return
			}
			ingestText(snippetFile, ask_db.DB)

		},
	}
)

func lintPass(rawText string, yamlData string, snippetFile string, yamlFile string, db *sql.DB) (lintStatus bool) {
	// Lint a snippet
	debug.Print("[*] Started linting.\n")
	// Need to check a few things for the linter:
	// variables in the metadata file and the snippet match up
	// the metadata file has all the necessary fields
	// the version numbers are ok
	var yamlStruct structs.Snippet
	yaml.Unmarshal([]byte(yamlData), &yamlStruct)

	// Extract variables
	pattern := regexp.MustCompile(`\{\{ ([a-zA-Z0-9_]+) \}\}`)
	matches := pattern.FindAllStringSubmatch(rawText, -1)
	variables := make(map[string]bool)
	for _, match := range matches {
		variables[match[1]] = true
	}

	SnippetVersion := yamlStruct.Version
	yamlVariables := make(map[string]bool)
	for key := range yamlStruct.Variables {
		yamlVariables[key] = true
	}

	// Compare to yaml file
	var missingInYaml []string
	for key := range variables {
		if yamlVariables[key] != true {
			missingInYaml = append(missingInYaml, key)
		}
	}

	var missingInText []string
	for key := range yamlVariables {
		if variables[key] != true {
			missingInText = append(missingInText, key)
		}
	}

	latestVersion, err := getLatestVersion(yamlStruct.Name, db)
	if err != nil {
		log.Println("[!] Error checking the latest version")
	}

	updatePossible, err := updatingVersion(yamlStruct.Version, latestVersion)
	if err != nil {
		log.Println("[!] Unable to tell if we can update")
	}

	if !updatePossible {
		latestVersionString := fmt.Sprintf("%d.%d.%d", latestVersion.Major, latestVersion.Minor, latestVersion.Patch)
		fmt.Fprintf(os.Stdout, "Proposed version is too low: %s vs %s\n", yamlStruct.Version, latestVersionString)
	}

	// If fail, open them both up
	if len(missingInText) != 0 || len(missingInYaml) != 0 || !updatePossible {
		fmt.Fprintf(os.Stdout, "[!] Linting failed.\n")
		// send to lint fixer

		SnippetText, yamlString := editForm(yamlStruct, snippetFile, SnippetVersion, db)
		var snippetData structs.Snippet
		yaml.Unmarshal([]byte(yamlString), &snippetData)
		snippetData.SnippetText = SnippetText
		snippetData.SnippetFile = snippetFile
		snippetYamlHandle := writeYaml(snippetData, yamlFile)
		snippetYamlHandle.Close()

		// write raw text to snippetFile
		os.WriteFile(snippetFile, []byte(snippetData.SnippetText), 0644)
		return true
	}
	debug.Print("[+] Linting complete.\n")
	return false
}

func getLatestVersion(snippetName string, db *sql.DB) (latestVersion structs.Version, err error) {
	//Get the latet version of a given snippet name
	debug.Print("[*] Getting latest version of snippet: %s", snippetName)

	if checkSnippetExists(db, snippetName) == false {
		debug.Print("[!] Snippet does not exist -- setting version number to 0.1.0: ", snippetName)
		return structs.Version{Major: 0, Minor: 1, Patch: 0}, nil
	}

	query := `SELECT version FROM snippets WHERE name = ? ORDER BY version DESC LIMIT 1`
	row := db.QueryRow(query, snippetName)

	var versionString string
	if err := row.Scan(&versionString); err != nil {
		return structs.Version{}, err
	}

	parts := strings.Split(versionString, ".")
	if len(parts) != 3 {
		return structs.Version{}, fmt.Errorf("invalid version format: %s", versionString)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return structs.Version{}, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return structs.Version{}, err
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return structs.Version{}, err
	}

	return structs.Version{Major: major, Minor: minor, Patch: patch}, nil
}

func versionBump(version string, updateType string) (newVersion string, err error) {
	// based on the update type, change the version string
	parts := strings.Split(version, ".")

	if len(parts) != 3 {
		panic("invalid version format")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		panic(err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		panic(err)
	}

	snippetVersion := structs.Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}

	switch updateType {
	case "Major":
		major = major + 1
		newVersion = fmt.Sprintf("%d.%d.%d", major, 0, 0)
	case "Minor":
		minor = minor + 1
		newVersion = fmt.Sprintf("%d.%d.%d", snippetVersion.Major, minor, 0)
	case "Patch":
		patch = patch + 1
		newVersion = fmt.Sprintf("%d.%d.%d", snippetVersion.Major, snippetVersion.Minor, patch)
	}
	return newVersion, nil
}

func updateDatabase(yamlData structs.Snippet, db *sql.DB) (err error) {
	//Add snippet to database
	debug.Print("Attempting to update database for %s v%s", yamlData.Name, yamlData.Version)

	//	yamlData, _ := ReadYAML[structs.Snippet](yamlFile)
	stmt, err := db.Prepare("INSERT INTO snippets (name, description, variables, version, snippetText) VALUES (?,?,?,?,?)")
	variables, _ := json.Marshal(yamlData.Variables)

	// read snippet text

	if yamlData.SnippetText == "" {
		file, err := os.Open(yamlData.SnippetFile)
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := ioutil.ReadAll(file)
		yamlData.SnippetText = string(byteValue)
	}

	_, err = stmt.Exec(yamlData.Name, yamlData.Description, variables, yamlData.Version, yamlData.SnippetText)
	if err != nil {
		return err
	}

	// TODO update tags

	return nil
}

func updatingVersion(version string, latestVersion structs.Version) (updatePossible bool, err error) {
	//think of this function as "should we allow an update"
	//convert version to Version struct
	//	fmt.Println(latestVersion)

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, err
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, err
	}

	if major > latestVersion.Major {
		return true, nil
	}
	if minor > latestVersion.Minor && major == latestVersion.Major {
		return true, nil
	}
	if patch > latestVersion.Patch && major == latestVersion.Major && minor == latestVersion.Minor {
		return true, nil
	}
	if version == "0.1.0" {
		return true, nil
	}
	return false, nil
}
func writeYaml(snippetStruct structs.Snippet, yamlFile string) (fileHandle *os.File) {
	//write a yaml file for a snippet

	tempYamlHandle, err := os.OpenFile(yamlFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer tempYamlHandle.Close()

	//  Marshall structs into yaml
	err = yaml.NewEncoder(tempYamlHandle).Encode(snippetStruct)
	if err != nil {
		log.Fatal(err)
	}

	return tempYamlHandle
}

// ReadYAML file amd return as a struct
func ReadYAML[T any](file string) (T, error) {
	// read a yaml file and return a struct
	var out T
	fileHandle, err := os.ReadFile(file)
	if err != nil {
		return out, err
	}

	err = yaml.Unmarshal(fileHandle, &out)
	if err != nil {
		return out, err
	}

	return out, nil

}

func snippetWizard(snippetStruct structs.Snippet, _ *sql.DB) (SnippetStruct structs.Snippet) {
	//  Find all the existing tags
	// var options []huh.Option[string]
	// existingTags, err := getTags(db)
	// for _, tag := range existingTags {
	// newOptions := huh.NewOption(tag, tag)
	// options = append(options, newOptions)
	// }
	newTagString := ""
	//  Get the global fields ready.  Global meaning we will definitely run this, whether or not we have any variable info to fill out
	var globalFields []huh.Field
	snippetNameField := huh.NewInput().
		Title("Snippet Name").
		Description("What is the name of the snippet?").
		Value(&snippetStruct.Name).
		Placeholder("snippetName").
		//  Provide a default value and only allow alphanumeric characters and _.
		// Also, check to make sure that name does not already exist.
		Validate(func(str string) error {
			if str == "" {
				return errors.New("Snippet name cannot be blank")
				// add that the snippet cannot just be numbers
			} else if !regexp.MustCompile(`^[a-zA-Z0-9_.]*$`).MatchString(str) {
				return errors.New("Snippet name can only contain alphanumeric characters, period or underscore")
			} else if str == "vscode" || str == "ultisnips" || str == "text" {
				return errors.New("Snippet cannot be named vscode, ultisnips, or text")
			} else if regexp.MustCompile(`^[0-9]*$`).MatchString(str) {
				return errors.New("Snippet cannot be only numbers")
			}

			return nil
		})
	snippetDescField := huh.NewText().
		Editor("vim").
		Title("What does the snippet do for the operator?").
		Lines(15).
		Placeholder("This snippet will allow the operator to establish port redirection to throw EternalBlue through a compromised toaster oven, then clean up logs.").
		Value(&snippetStruct.Description).
		Validate(func(str string) error {
			if len(str) > 255 {
				return fmt.Errorf("Description cannot exceed 255 characters. %s > 255", strconv.Itoa(len(str)))
			}
			return nil
		},
		)

	//  Tags make the form kind of crazy looking. I don't love that.  Maybe something from bubble tea instead.
	//  also could add optional "do you want to add tags?"
	// snippetTagField := huh.NewMultiSelect[string]().
	// Title("Tags").
	// Description("Select any existing tags you want associated with the snippet.\nUse / to search and <SPACE> to select.").
	// Options(options...).
	// Value(&snippetStruct.Tags)
	/*
		snippetNewTagsField := huh.NewInput().
			Title("New tags").
			Description("Add any new tags you want added to the database.  Comma delimit for multiple tags.").
			Value(&newTagString)
	*/

	globalFields = append(globalFields, snippetNameField)
	globalFields = append(globalFields, snippetDescField)
	// globalFields = append(globalFields, snippetTagField)
	//globalFields = append(globalFields, snippetNewTagsField)

	//  Loop through the variables (if we have them) and build the form for them
	var varFields []huh.Field
	for variable := range snippetStruct.Variables {
		tempVarStruct := snippetStruct.Variables[variable]
		snippetStruct.Variables[variable] = tempVarStruct
		descriptionPrompt := fmt.Sprintf("What is %s and how do you obtain this value?", variable)
		examplePrompt := fmt.Sprintf("What is an example of %s?", variable)
		defaultPrompt := fmt.Sprintf("What are default values of %s? Use | to delimit options", variable)
		fieldText := huh.NewText().
			Editor("vim").
			Lines(15).
			Title(descriptionPrompt).
			Placeholder("This is a description of the variable.  How you get it or why it matters.").
			//Is there a bug here?  When I enter the editor to change the value, the result is assigned for all of the variables in the group, not just the specific variable.
			Value(&tempVarStruct.Description).
			Validate(func(str string) error {
				if len(str) > 255 {
					return fmt.Errorf("Description cannot exceed 255 characters. %s > 255", strconv.Itoa(len(str)))
				}
				return nil
			},
			)

		varFields = append(varFields, fieldText)
		field := huh.NewInput().
			Title("Example values").
			Description(examplePrompt).
			Value(&tempVarStruct.ExampleValue)
		varFields = append(varFields, field)
		field = huh.NewInput().
			Title("Default values").
			Description(defaultPrompt).
			Value(&tempVarStruct.DefaultValue)
		varFields = append(varFields, field)
		snippetStruct.Variables[variable] = tempVarStruct
	}

	//  We get different fields if we do not have any variables to fill out
	var form *huh.Form
	if len(snippetStruct.Variables) > 0 {
		form = huh.NewForm(
			huh.NewGroup(globalFields...),
			huh.NewGroup(varFields...),
		)
	} else {
		form = huh.NewForm(
			huh.NewGroup(globalFields...),
		)
	}

	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}
	//  Add new tags
	if len(snippetStruct.Tags) == 0 {
		newTags := strings.Split(newTagString, ",")
		for _, tag := range newTags {
			snippetStruct.Tags = append(snippetStruct.Tags, tag)
		}
	}

	return snippetStruct
}

func findVariables(snippetText string) (variables map[string]bool, err error) {
	//look through text to identify variables

	pattern := regexp.MustCompile(`\{\{ ([a-zA-Z0-9_]+) \}\}`)
	matches := pattern.FindAllStringSubmatch(snippetText, -1)
	variables = make(map[string]bool)
	for _, match := range matches {
		variables[match[1]] = true
	}
	return variables, nil
}

func snippetFileToName(snippetFile string) (snippetName string) {
	// get the basename
	var baseName string
	i := strings.LastIndex(snippetFile, "/")
	if i != -1 {
		baseName = snippetFile[i+1:]
	} else {
		baseName = snippetFile
	}

	// strip of the extension, if it exists
	i = strings.LastIndex(baseName, ".")
	if i != -1 {
		snippetName = baseName[:i]
	} else {
		snippetName = baseName
	}
	return snippetName
}

func checkSnippetCollision(db *sql.DB, snippetName string, snippetVersion string) bool {
	//See if a collision exists if we ingest.  This function answers the question "Does the name and version of this snippet exist already?"
	debug.Print("[*] Checking if snippet name and version already exists: %sv%2", snippetName, snippetVersion)

	stmt, err := db.Prepare("SELECT version FROM snippets WHERE name =?")

	if err != nil {
		panic(err)
	}
	rows, err := stmt.Query(snippetName)
	if err != nil {
		return false
	}

	var existingVersion string
	for rows.Next() {
		err := rows.Scan(&existingVersion)
		if err != nil {
			log.Fatal(err)
		}
		if existingVersion == snippetVersion {
			return true
		}
	}

	//If we have gone through all the rows and the version number still is not there, we can update safely
	return false
}

func checkSnippetExists(db *sql.DB, snippetName string) bool {
	//See if a snippet id or name is valid
	debug.Print("[*] Checking if snippet exists: %s", snippetName)

	var name string

	if _, parseErr := strconv.Atoi(snippetName); parseErr == nil {
		debug.Print("[!] Looks like a snippet id")
		err := db.QueryRow("SELECT name FROM snippets WHERE id = ?", snippetName).Scan(&name)
		if err != nil {
			if err == sql.ErrNoRows {
				return false
			}
			log.Println("Error querying by name:", err)
			return false
		}
		return true
	}
	debug.Print("[!] Looks like a snippet name: %s", snippetName)
	err := db.QueryRow("SELECT name FROM snippets WHERE LOWER(name) = LOWER(?) LIMIT 1", snippetName).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Println("Error querying by name:", err)
		return false
	}
	return true
}

func ingestZip(snippetFile string, db *sql.DB) error {
	//extract the snippets from a zip file and ingest each one

	debug.Print("Attempting to extract contents of zip archive for the database: %s", snippetFile)
	//extract contents

	zipReader, err := zip.OpenReader(snippetFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	addedCount := 0
	for _, yamlFile := range zipReader.File {
		debug.Print("Processing %s", yamlFile.Name)
		fileReader, err := yamlFile.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		debug.Print("Opened %s", yamlFile.Name)

		yamlBytes, err := io.ReadAll(fileReader)
		if err != nil {
			return err
		}

		debug.Print("Marshalling data: %s", yamlFile.Name)
		var yamlData structs.Snippet
		err = yaml.Unmarshal(yamlBytes, &yamlData)
		if err != nil {
			return err
		}

		if checkSnippetCollision(db, yamlData.Name, yamlData.Version) {
			fmt.Fprintf(os.Stdout, "[!] Snippet already exists in database: %sv%s\n", yamlData.Name, yamlData.Version)
			fmt.Fprintf(os.Stdout, "[*] Perhaps you would like to update it:\n")
			fmt.Fprintf(os.Stdout, "\task edit %s", yamlData.Name)
			fmt.Fprintf(os.Stdout, "[*] Or remove it:\n")
			fmt.Fprintf(os.Stdout, "\task remove %s", yamlData.Name)
			continue
		}

		err = updateDatabase(yamlData, db)
		if err != nil {
			return nil
		}
		addedCount++
	}

	fmt.Fprintf(os.Stdout, "[+] Snippet archive added: %s\n", snippetFile)
	fmt.Fprintf(os.Stdout, "[+] Snippets added: %s\n", strconv.Itoa(addedCount))
	return nil
}

func ingestYaml(snippetFile string, db *sql.DB) error {
	//create entry based on yaml file
	debug.Print("[*] Attempting to ingest yaml file: %s", snippetFile)

	//open up the yaml file
	yamlFile, err := os.Open(snippetFile)
	if err != nil {
		return err
	}
	defer yamlFile.Close()

	yamlData, _ := ReadYAML[structs.Snippet](yamlFile.Name())

	//check if the name and version of the snippet exists. If not, then we can add it
	if checkSnippetCollision(db, yamlData.Name, yamlData.Version) {
		fmt.Fprintf(os.Stdout, "[!] Snippet already exists in database: %sv%s\n", yamlData.Name, yamlData.Version)
		fmt.Fprintf(os.Stdout, "[*] Perhaps you would like to update it:\n")
		fmt.Fprintf(os.Stdout, "\task edit %s", yamlData.Name)
		fmt.Fprintf(os.Stdout, "[*] Or remove it:\n")
		fmt.Fprintf(os.Stdout, "\task remove %s", yamlData.Name)

		return nil
	}

	debug.Print("[*] Ingesting snippet : %sv%s", yamlData.Name, yamlData.Version)

	err = updateDatabase(yamlData, db)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "[+] Snippet added: %sv%s\n", yamlData.Name, yamlData.Version)
	return nil
}

func ingestText(snippetFile string, db *sql.DB) error {
	//If it is not a yaml file, then

	var snippetName string
	snippetName = snippetFileToName(snippetFile)

	if checkSnippetExists(db, snippetName) == true {
		fmt.Fprintf(os.Stdout, "[!] Snippet already exists in database: %s\n", snippetName)
		fmt.Fprintf(os.Stdout, "[*] Perhaps you would like to update it:\n")
		fmt.Fprintf(os.Stdout, "\task edit %s\n", snippetName)
		fmt.Fprintf(os.Stdout, "[*] Or remove it:\n")
		fmt.Fprintf(os.Stdout, "\task remove %s", snippetName)

		return nil
	}
	debug.Print("[+] Adding new snippet: %s\n", snippetName)

	file, err := os.Open(snippetFile)
	if err != nil {
		log.Fatal(err)
		return err
	}



	defer file.Close()

	byteValue, _ := io.ReadAll(file)
	rawText := string(byteValue)

	// parse snippet to grab the variables
	// Variables are enclosed in double curly braces with a space on each side.  The variable name is alphanumeric with underscores allowed.

	// snippet wizard
	newSnippet := structs.Snippet{
		Name:      snippetName,
		Variables: make(map[string]*structs.Variable),
		Version:   "0.1.0",
		Tags:      []string{},
	}

	variables, err := findVariables(rawText)
	// process variables to be a map with empty fields

	//Escape any escape characters
	
	escaped := strings.ReplaceAll(string(rawText), `\`, `\\`)


	tmpFile, err := os.CreateTemp("", snippetName + "temp-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(escaped))
	if err != nil {
		return err
	}

	tmpFile.Close()



	Variables := make(map[string]*structs.Variable, len(variables))
	for key := range variables {
		Variables[key] = &structs.Variable{}
	}
	// add it to the new struct
	newSnippet.Variables = Variables

	newSnippet = snippetWizard(newSnippet, db)

	// write the yaml
	newSnippet.SnippetFile = tmpFile.Name()
	newSnippet.Version = "0.1.0"

	//  Create temp file
	tempYamlHandle, err := os.CreateTemp("", "temp-*.yaml")
	if err != nil {
		log.Fatal(err)
	}

	yamlHandle := writeYaml(newSnippet, tempYamlHandle.Name())
	yamlFile := yamlHandle.Name()

	yamlData, _ := ReadYAML[structs.Snippet](yamlFile)
	yamlNew, _ := yaml.Marshal(yamlData)
	yamlString := string(yamlNew)

	// Linter
	if lintPass(rawText, yamlString, snippetFile, yamlFile, db) != false {
		log.Printf("[!] Lint failed.\n")
	}

	// Update

	//yamlData, _ := ReadYAML[structs.Snippet](yamlHandle.Name())
	updateDatabase(yamlData, db)

	// Close up the database and the temp file
	yamlHandle.Close()

	os.Remove(yamlHandle.Name())

	return nil
}

func init() {
}
