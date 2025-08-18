package cmd

import (
	"ask/ask_db"
	"ask/internal/debug"
	structs "ask/internal/model"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	yaml "gopkg.in/yaml.v3"

	//"github.com/jedib0t/go-pretty/v6/text"

	"golang.design/x/clipboard"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// OutputDir is a flag variable
var OutputDir string

// RenderAll is a flag variable
var RenderAll bool

// given a snippet id, return the relevant snippet data
func getSnippetData(snippetID string, db *sql.DB) (snippetData structs.Snippet, err error) {
	var jsonVar string
	stmt, err := db.Prepare("SELECT name, version, description, variables, snippetText FROM snippets WHERE id =?")

	if err != nil {
		return snippetData, err
	}
	rows, err := stmt.Query(snippetID)
	if err != nil {
		return snippetData, err
	}

	for rows.Next() {
		err := rows.Scan(&snippetData.Name, &snippetData.Version, &snippetData.Description, &jsonVar, &snippetData.SnippetText)
		if err != nil {
			log.Fatal(err)
			return snippetData, err
		}
	}
	// convert json string to Variables struct

	json.Unmarshal([]byte(jsonVar), &snippetData.Variables)

	return snippetData, nil
}

func render(snippetID string, db *sql.DB) error {
	//render a snippet based on the id
	debug.Print("[*] Attempting to render %s", snippetID)

	snippetStruct, err := getSnippetData(snippetID, db)
	if err != nil {
		return err
	}
	if LoadFile == "" {
		var varFields []huh.Field
		for variable := range snippetStruct.Variables {
			tempVarStruct := snippetStruct.Variables[variable]
			snippetStruct.Variables[variable] = tempVarStruct
			field := huh.NewInput().
				Title(variable).
				Description(snippetStruct.Variables[variable].Description + "\nExample value: " + snippetStruct.Variables[variable].ExampleValue).
				Value(&snippetStruct.Variables[variable].DefaultValue)
			varFields = append(varFields, field)
		}
		if len(snippetStruct.Variables) > 0 {
			form := huh.NewForm(
				huh.NewGroup(varFields...),
			)
			err := form.Run()
			if err != nil {
				log.Fatal(err)
			}
			//save the output, replace in the text and then render

			replaceMap := make(map[string]string)
			var yamlLine string
			for variable := range snippetStruct.Variables {
				value := snippetStruct.Variables[variable].DefaultValue
				replaceMap["{{ "+variable+" }}"] = value
				if VarSave == true {
					//I anticipate that if the value has single or double quotes, this line could become problematic
					yamlLine = yamlLine + variable + ": '" + value + "'\n"
				}

			}

			for variable, value := range replaceMap {
				snippetStruct.SnippetText = strings.ReplaceAll(snippetStruct.SnippetText, variable, value)

			}

			if VarSave == true {
				time := time.Now().Format("2006-01-02_15-04-05")
				fileName := time + "_" + snippetStruct.Name + ".yaml"
				err = os.WriteFile(fileName, []byte(yamlLine), 0644)
				if err != nil {
					return err
				}
			}

		}
	} else {
		yamlFile, err := os.Open(LoadFile)
		if err != nil {
			return err
		}
		defer yamlFile.Close()

		yamlData, err := io.ReadAll(yamlFile)
		if err != nil {
			return err
		}

		var replaceMap map[string]string
		err = yaml.Unmarshal(yamlData, &replaceMap)
		if err != nil {
			return err
		}

		for variable, value := range replaceMap {
			snippetStruct.SnippetText = strings.ReplaceAll(snippetStruct.SnippetText, "{{ "+variable+" }}", value)
		}

	}

	if Clipboard {
		err := clipboard.Init()
		if err != nil {
			panic(err)
		}
		clipboard.Write(clipboard.FmtText, []byte(snippetStruct.SnippetText))
		fmt.Fprintf(os.Stdout, "[+] Copied snippet to clipboard: %s\n", snippetStruct.Name)
	} else {

		fmt.Fprintf(os.Stdout, "%s\n", snippetStruct.SnippetText)
	}

	return nil

}

// Clipboard is a flag variable
var Clipboard bool

// VarSave is a flag variable
var VarSave bool

// LoadFile is a flag variable
var LoadFile string

var (
	renderCmd = &cobra.Command{
		Use:   "render",
		Short: "Render snippets",
		Long: `Extract snippets from the database.  You can either extract a single snippet from the database using the snippet ID or name for immediate use, or you can render the entire database to a target directory in text, ultisnip or vscode snippet formats.

EXAMPLES
ask render example
	#You will then be prompted to fill out the variables for the example snippet

ask render ultisnips --outdir ~/.config/nvim/UltiSnips/all
	#This will convert the database to UltiSnips format
`,

		Args: cobra.MinimumNArgs(0),
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 0 {
				snippetID, _ := listSnippets(ask_db.DB)
				err := render(snippetID, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[!] error rendering snippet: %w\n", err)
				}

			} else {
				snippetString := args[0]
				//get snippet id or name
				snippetID, _, err := getSnippetID(snippetString, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[!] error getting snippet name and id: %w\n", err)
				}

				err = render(snippetID, ask_db.DB)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[!] error rendering snippet: %w\n", err)
				}

			}

		},
	}
)

func processText(snippetID string, db *sql.DB) (processed bool, err error) {
	debug.Print("Attempting text render for snippet id: %s", snippetID)
	// Grab all the information for the snippet
	snippetData, err := getSnippetData(snippetID, db)
	if err != nil {
		return false, err
	}

	//Check if it is the latest version.  If it is not and the flag for render all is not set, return

	if !isLatestSnippet(db, snippetID, snippetData.Name) && RenderAll != true {
		debug.Print("[!] Not the latest version, and RenderAll flag not set.  Skipping: %s", snippetData.Name)
		return false, nil
	}

	debug.Print("Found information for snippet: %s", snippetData.Name)

	var headerSection, variableSection, bodySection string
	snippetFileName := fmt.Sprintf("%s_v%s.txt", snippetData.Name, snippetData.Version)

	headerSection = "--------------Snippet Info ----------------\n"
	// Create the header/metadata section

	var headerTable bytes.Buffer
	tw := table.NewWriter()
	tw.SetOutputMirror(&headerTable)
	tw.AppendHeader(table.Row{"Name", "Version", "Description"})
	tw.AppendRow(table.Row{snippetData.Name, snippetData.Version, snippetData.Description})
	tw.SetStyle(table.StyleRounded) // Optional: You can use StyleLight, StyleBold, etc.
	tw.Render()

	// Merge everything into headerSection
	headerSection += headerTable.String()

	//If there are no variables, skip that section
	if len(snippetData.Variables) > 0 {
		debug.Print("The '%s_v%s' snippet has variables.  Processing them now", snippetData.Name, snippetData.Version)
		variableSection = "--------------Variable Section-------------\n"
		for variable, variableInfo := range snippetData.Variables {
			variableSection = fmt.Sprintf("%s#%s description - %s\n", variableSection, variable, variableInfo.Description)
			if variableInfo.DefaultValue != "" {
				variableSection = fmt.Sprintf("%ssed -i 's/{{ %s }}/%s/g' %s\n", variableSection, variable, variableInfo.DefaultValue, snippetFileName)
			} else {
				variableSection = fmt.Sprintf("%ssed -i 's/{{ %s }}/YOUR_VALUE_FOR_%s/g' %s\n", variableSection, variable, variable, snippetFileName)
			}
		}
	}

	bodySection = "--------------Snippet Body ----------------\n"
	bodySection = fmt.Sprintf("%s\n%s", bodySection, snippetData.SnippetText)

	//bring it all together and write to file
	fullText := headerSection + variableSection + bodySection

	err = writeSnippet(fullText, snippetFileName)
	if err != nil {
		return false, err
	}
	return true, nil
}

func writeSnippet(snippetText string, snippetFileName string) (err error) {

	//Change ~ to home drive because golang does not like it
	//This could be an issue if some weirdo wanted to do something like ../~/.config/../../etc
	if OutputDir[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		OutputDir = strings.Replace(OutputDir, "~", homeDir, 1)
	}

	//Create the output directory if it does not exist
	debug.Print("[-] Attempting to create directory: %s", OutputDir)

	// Check if the directory exists
	_, err = os.Stat(OutputDir)
	if os.IsNotExist(err) {
		// Directory does not exist, so create it and all parent directories
		err := os.MkdirAll(OutputDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		debug.Print("Directory created:", OutputDir)
	}

	err = os.WriteFile(OutputDir+"/"+snippetFileName, []byte(snippetText), 0644)
	if err != nil {
		return err
	}

	debug.Print("[+] Wrote snippet: %s", snippetFileName)
	return nil
}

func processVSCode(snippetID string, db *sql.DB) (processed bool, err error) {
	debug.Print("[*] Attempting VSCode render for snippet id: %s", snippetID)
	// Grab all the information for the snippet
	snippetData, err := getSnippetData(snippetID, db)
	if err != nil {
		return false, err
	}
	debug.Print("[+] Found information for snippet: %s", snippetData.Name)

	if !isLatestSnippet(db, snippetID, snippetData.Name) && RenderAll != true {
		debug.Print("[*] Not the latest version, and RenderAll flag not set.  Skipping: %s", snippetData.Name)
		return false, nil
	}

	var headerSection string //, variableSection, bodySection string
	//snippetFileName := fmt.Sprintf("%s_v%s.json", snippetData.Name, snippetData.Version)

	// Create the header/metadata section
	// Figure out the format for VS code rendering

	/*
		"example": {
		  "prefix": "trigger",
		  "description": "example"
		  "body": [
		    "this is my snippet"
		  ],
		}
	*/

	headerSection = fmt.Sprintf("\n\"%s_v%s\": {\n\t \"prefix\": \"%s\",\n\t \"description\": \"%s\"\n\t \"body\": [\n", snippetData.Name, snippetData.Version, snippetData.Name, snippetData.Description)

	var bodyTable bytes.Buffer
	tw := table.NewWriter()
	tw.SetOutputMirror(&bodyTable)
	tw.AppendHeader(table.Row{"Name", "Version", "Description"})
	tw.AppendRow(table.Row{snippetData.Name, snippetData.Version, snippetData.Description})
	tw.SetStyle(table.StyleRounded) // Optional: You can use StyleLight, StyleBold, etc.
	tw.Render()

	// Merge everything into bodySection
	metaSection := "--------------Snippet Info ----------------\n"
	bodySection := metaSection + bodyTable.String()

	var variableSection string
	var variableKey []string
	//If there are no variables, skip that section
	if len(snippetData.Variables) > 0 {
		debug.Print("[!] The '%s_v%s' snippet has variables.  Processing them now", snippetData.Name, snippetData.Version)
		variableSection = "--------------Variable Section-------------\n"

		var variableTable bytes.Buffer
		tw := table.NewWriter()
		tw.SetOutputMirror(&variableTable)
		tw.AppendHeader(table.Row{"Variable Name", "Value", "Example Value", "Description"})

		for variable, variableInfo := range snippetData.Variables {
			//The index for the variables is not set because maps are unordered.  If I want to have consistent orders for the variables, I may have to rethink my Variables struct.
			variableKey = append(variableKey, variable)
			if variableInfo.DefaultValue != "" {
				//Need to fix the index numbers here
				tw.AppendRow(table.Row{variable, "${" + strconv.Itoa(len(variableKey)*10) + ":" + variableInfo.DefaultValue + "}", variableInfo.ExampleValue, variableInfo.Description})
			} else {
				//Need to fix the index numbers here
				tw.AppendRow(table.Row{variable, "${" + strconv.Itoa(len(variableKey)*10) + "}", variableInfo.ExampleValue, variableInfo.Description})
			}
		}
		tw.SetStyle(table.StyleRounded)
		tw.Render()
		variableSection += variableTable.String()

	}

	//make variable substitutions
	//Do the ultisnips escapes
	snippetData.SnippetText = strings.ReplaceAll(snippetData.SnippetText, "`", "\\`")
	snippetData.SnippetText = strings.ReplaceAll(snippetData.SnippetText, "$", "\\$")

	//Replace the variables
	for index, key := range variableKey {
		debug.Print("[*] Replacing {{ %s }} with $%s", key, strconv.Itoa((index+1)*10))
		snippetData.SnippetText = strings.ReplaceAll(snippetData.SnippetText, "{{ "+key+" }}", "$"+strconv.Itoa((index+1)*10))
	}

	//enclose in quotes

	var processedBody string
	snippetData.SnippetText = "--------------Snippet Text-----------------\n" + snippetData.SnippetText

	for _, line := range strings.Split(bodySection+variableSection+snippetData.SnippetText, "\n") {
		//line = strings.ReplaceAll(line, "$", "\\$")
		line = strings.ReplaceAll(line, "\\", "\\\\")
		line = strings.ReplaceAll(line, "\"", "\\\"")

		processedBody += fmt.Sprintf("\"%s\",\n", line)
	}

	processedBody = processedBody + "]\n"
	fullText := headerSection + processedBody + "}"

	snippetFileName := fmt.Sprintf("%s_v%s.json", snippetData.Name, snippetData.Version)
	err = writeSnippet(fullText, snippetFileName)
	if err != nil {
		return false, err
	}

	return true, nil
}

func processUlti(snippetID string, db *sql.DB) (processed bool, err error) {
	debug.Print("[*] Attempting UltiSnips render for snippet id: %s", snippetID)
	// Grab all the information for the snippet
	snippetData, err := getSnippetData(snippetID, db)
	if err != nil {
		return false, err
	}
	debug.Print("[+] Found information for snippet: %s", snippetData.Name)

	if !isLatestSnippet(db, snippetID, snippetData.Name) && RenderAll != true {
		debug.Print("[!] Not the latest version, and RenderAll flag not set.  Skipping: %s", snippetData.Name)
		return false, nil
	}

	var headerSection, variableSection, bodySection string
	snippetFileName := fmt.Sprintf("%s_v%s.snippets", snippetData.Name, snippetData.Version)

	headerSection = fmt.Sprintf("snippet %s \"%s\"\n\n", snippetData.Name, snippetData.Description)

	// Table title and setup
	metaTable := "--------------Snippet Information----------\n"

	var headerTable bytes.Buffer
	tw := table.NewWriter()
	tw.SetOutputMirror(&headerTable)
	tw.AppendHeader(table.Row{"Name", "Version", "Description"})
	tw.AppendRow(table.Row{snippetData.Name, snippetData.Version, snippetData.Description})
	tw.SetStyle(table.StyleRounded) // Optional: You can use StyleLight, StyleBold, etc.
	tw.Render()

	// Merge everything into headerSection
	metaTable += headerTable.String()
	headerSection += metaTable

	var variableKey []string

	//If there are no variables, skip that section
	if len(snippetData.Variables) > 0 {
		debug.Print("[*] The '%s_v%s' snippet has variables.  Processing them now", snippetData.Name, snippetData.Version)
		variableSection = "--------------Variable Section-------------\n"

		var variableTable bytes.Buffer
		tw := table.NewWriter()
		tw.SetOutputMirror(&variableTable)
		tw.AppendHeader(table.Row{"Variable Name", "Value", "Example Value", "Description"})

		for variable, variableInfo := range snippetData.Variables {
			//The index for the variables is not set because maps are unordered.  If I want to have consistent orders for the variables, I may have to rethink my Variables struct.
			variableKey = append(variableKey, variable)
			if variableInfo.DefaultValue != "" {
				//Need to fix the index numbers here
				tw.AppendRow(table.Row{variable, "${" + strconv.Itoa(len(variableKey)*10) + ":" + variableInfo.DefaultValue + "}", variableInfo.ExampleValue, variableInfo.Description})
			} else {
				//Need to fix the index numbers here
				tw.AppendRow(table.Row{variable, "${" + strconv.Itoa(len(variableKey)*10) + "}", variableInfo.ExampleValue, variableInfo.Description})
			}
		}
		tw.SetStyle(table.StyleRounded)
		tw.Render()
		variableSection += variableTable.String()

	}

	bodySection = "--------------Snippet Body ----------------\n"

	//Do the ultisnips escapes
	snippetData.SnippetText = strings.ReplaceAll(snippetData.SnippetText, "`", "\\`")
	snippetData.SnippetText = strings.ReplaceAll(snippetData.SnippetText, "$", "\\$")

	//Replace the variables
	for index, key := range variableKey {
		debug.Print("[*] Replacing {{ %s }} with $%s", key, strconv.Itoa((index+1)*10))
		snippetData.SnippetText = strings.ReplaceAll(snippetData.SnippetText, "{{ "+key+" }}", "$"+strconv.Itoa((index+1)*10))
	}

	bodySection = fmt.Sprintf("%s$0%s", bodySection, snippetData.SnippetText)

	//Create the snippet body section

	//bring it all together and write to file
	fullText := headerSection + variableSection + bodySection + "\nendsnippet"

	//Create the output directory if it does not exist
	debug.Print("[-] Attempting to create directory: %s.", OutputDir)

	err = writeSnippet(fullText, snippetFileName)
	if err != nil {
		return false, err
	}
	return true, nil
}

func vscodeRender(db *sql.DB) (processedSnippets int, err error) {
	debug.Print("[*] Starting vscode render function")
	// Loop through the entire database and render each one as a snippet file
	var id string
	var allSnippetIDs []string

	rows, err := db.Query("SELECT id FROM snippets")
	if err != nil {
		log.Fatalf("[!] Query failed: %v", err)
	}

	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("Row scan failed: %v", err)
		}
		debug.Print("[+] Found snippet: %s", id)
		allSnippetIDs = append(allSnippetIDs, id)
	}

	for _, snippetID := range allSnippetIDs {
		debug.Print("[*] Processing snippet with id: %s", snippetID)
		processed, err := processVSCode(snippetID, ask_db.DB)
		if err != nil {
			return processedSnippets, err
		}
		if processed == true {
			processedSnippets++
		}
	}

	return processedSnippets, nil
}

func ultiRender(db *sql.DB) (processedSnippets int, err error) {
	debug.Print("[*] Starting ultiRender function")
	// Loop through the entire database and render each one as a snippet file
	var id string
	var allSnippetIDs []string

	rows, err := db.Query("SELECT id FROM snippets")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("Row scan failed: %v", err)
		}
		debug.Print("[+] Found snippet: %s", id)
		allSnippetIDs = append(allSnippetIDs, id)
	}

	for _, snippetID := range allSnippetIDs {
		debug.Print("[*] Processing snippet with id: %s", snippetID)
		processed, err := processUlti(snippetID, ask_db.DB)
		if err != nil {
			return processedSnippets, err
		}
		if processed == true {
			processedSnippets++
		}
	}

	return processedSnippets, nil
}

func textRender(db *sql.DB) (processedSnippets int, err error) {
	debug.Print("[*] Starting textRender function")
	// Loop through the entire database and render each one as a snippet file
	var id string
	var allSnippetIDs []string

	rows, err := db.Query("SELECT id FROM snippets")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("Row scan failed: %v", err)
		}
		debug.Print("[+] Found snippet: %s", id)
		allSnippetIDs = append(allSnippetIDs, id)
	}

	for _, snippetID := range allSnippetIDs {
		debug.Print("[*] Processing snippet with id: %s", snippetID)
		processed, err := processText(snippetID, ask_db.DB)
		if err != nil {
			return processedSnippets, err
		}
		if processed == true {
			processedSnippets++
		}
	}

	return processedSnippets, nil
}

var (
	textConvertCmd = &cobra.Command{
		Use:   "text",
		Short: "Render to simple text files for manual modification",
		Long:  `Extract highest version snippet from the database and render it as a generic text document.  Note that multiple default values for variables is not supported for text snippets, and the provided default value, if used, will be created in the resulting sed command.`,
		//Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, _ []string) {
			processedSnippets, err := textRender(ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Failed to render text snippets: %w", err)
				return
			}
			fmt.Fprintf(os.Stdout, "[+] Text snippets rendered to %s: %s", OutputDir, strconv.Itoa(processedSnippets))
		},
	}
)
var (
	ultisnipsConvertCmd = &cobra.Command{
		Use:   "ultisnips",
		Short: "Render to ultisnips files for use in vim or nvim",
		Long:  `Extract highest version snippet from the database and render it as an Ultisnips snippet.`,
		//Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, _ []string) {
			processedSnippets, err := ultiRender(ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Failed to render ultisnips snippets: %w", err)
				return
			}
			fmt.Fprintf(os.Stdout, "[+] Ultisnips snippets rendered to %s: %s", OutputDir, strconv.Itoa(processedSnippets))
		},
	}
)

var (
	vscodeConvertCmd = &cobra.Command{
		Use:   "vscode",
		Short: "Render to vscode snippet files",
		Long:  `Extract highest version snippet from the database and render it in a variety of formats.  Currently supports vscode, UltiSnips and VSCode format`,
		//Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, _ []string) {
			processedSnippets, err := vscodeRender(ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Failed to render ultisnips snippets: %w", err)
				return
			}
			fmt.Fprintf(os.Stdout, "[+] VSCode snippets rendered to %s: %s", OutputDir, strconv.Itoa(processedSnippets))
		},
	}
)

func init() {
	renderCmd.Flags().BoolVarP(&Clipboard, "clipboard", "x", false, "Render to clipboard.")
	renderCmd.AddCommand(textConvertCmd)
	renderCmd.PersistentFlags().StringVarP(&OutputDir, "outputdir", "o", "./snippets", "Where to output the files, reading from the flag first, then the config file.")
	renderCmd.PersistentFlags().StringVarP(&LoadFile, "load", "l", "", "YAML file to load variables for rendering")
	renderCmd.PersistentFlags().BoolVarP(&VarSave, "save", "s", false, "Save the variable selections in a yaml file for future use. Named with timestamp and the snippet name.")
	renderCmd.AddCommand(ultisnipsConvertCmd)
	renderCmd.AddCommand(vscodeConvertCmd)

	vscodeConvertCmd.Flags().BoolVarP(&RenderAll, "all", "a", false, "Render all versions, not just the latest")
	textConvertCmd.Flags().BoolVarP(&RenderAll, "all", "a", false, "Render all versions, not just the latest")
	ultisnipsConvertCmd.Flags().BoolVarP(&RenderAll, "all", "a", false, "Render all versions, not just the latest")

}
