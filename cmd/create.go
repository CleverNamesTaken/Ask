// add generation for the creation of the ask database

package cmd

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"ask/config"
	"io"
	"os"
	"strconv"

	//	"strings"
	//	"database/sql"
	"ask/ask_db"
	"ask/internal/debug"
	structs "ask/internal/model"
	"log"

	"github.com/spf13/cobra"
)

// Yaml is a flag value
var Yaml bool

// ArchiveFile is a flag value
var ArchiveFile string

func createYaml(snippetID string, db *sql.DB) (yamlFile string, err error) {
	//create yaml based on the snippet id
	debug.Print("[*] Attempting creation of yaml file for %s.", snippetID)

	var snippetData structs.Snippet
	var jsonVar string
	stmt, err := db.Prepare("SELECT name, version, description, variables, snippetText FROM snippets WHERE id =?")

	if err != nil {
		return yamlFile, err
	}
	rows, err := stmt.Query(snippetID)
	if err != nil {
		return yamlFile, err
	}

	for rows.Next() {
		err := rows.Scan(&snippetData.Name, &snippetData.Version, &snippetData.Description, &jsonVar, &snippetData.SnippetText)
		if err != nil {
			log.Fatal(err)
		}
	}
	// convert json string to Variables struct

	json.Unmarshal([]byte(jsonVar), &snippetData.Variables)

	//rename yaml to name version
	yamlName := fmt.Sprintf("%s_v%s.yaml", snippetData.Name, snippetData.Version)
	writeYaml(snippetData, yamlName)
	debug.Print("[+] Wrote to %s", yamlName)

	return yamlName, nil
}
func createZip(db *sql.DB) error {
	// Get all data from the database and export a zip file
	debug.Print("[*] Attempting creation of zip archive for the database.")

	//Query all the id numbers

	query := `
    SELECT id FROM snippets
    `
	rows, err := db.Query(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Error querying ids: %w\n", err)
		return err
	}
	defer rows.Close()

	var idSlice []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		idSlice = append(idSlice, id)
	}

	if len(idSlice) == 0 {
		fmt.Fprintf(os.Stdout, "[+] Database is empty\n")
		return nil
	}

	//for each id number, create a yaml
	var yamlFiles []string
	for _, id := range idSlice {
		yamlFile, err := createYaml(strconv.Itoa(id), db)
		if err != nil {
			log.Printf("[!] Failed to create yaml file ")
			fmt.Println("%w", err)
			return err
		}
		yamlFiles = append(yamlFiles, yamlFile)
	}

	//zip up all the yamls
	err = zipYamls(yamlFiles)
	if err != nil {
		log.Printf("Failed to create zip file ")
		fmt.Println("%w", err)
		return err
	}

	fmt.Printf("[+] Created zip archive: %s", ArchiveFile)

	//delete all the yaml files
	for _, filename := range yamlFiles {
		debug.Print("Removing %s", filename)
		os.Remove(filename)
	}

	return nil
}

func zipYamls(yamlFiles []string) error {
	//zip up all the yaml files
	debug.Print("Attempting creation of zip file.")
	zipfile, err := os.Create(ArchiveFile)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	zipwriter := zip.NewWriter(zipfile)
	defer zipwriter.Close()

	for _, filename := range yamlFiles {
		debug.Print("Adding %s to zip", filename)
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		//defer os.Remove filename
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Method = zip.Deflate

		writer, err := zipwriter.CreateHeader(header)
		if err != nil {
			return err
		}

		_, err = io.Copy(writer, file)
		if err != nil {
			return err
		}
	}

	err = zipwriter.Close()
	if err != nil {
		return err
	}

	debug.Print("Zip file created successfully.")
	return nil
}

var (
	archiveCmd = &cobra.Command{
		Use:   "zip",
		Short: "Create a zip archive of the database",
		Long: `Back up the snippets in your database or allow them to be transferred

EXAMPLES

ask create zip --outfile 2025_07_26_ask.zip
	#Backup all the ask data into the 2025_07_26_ask.zip file

ask add 2025_07_26_ask.zip
	#Ingest the data into a new database.
`,
		//Args:  cobra.MinimumNArgs(1),
		Run: func(_ *cobra.Command, _ []string) {
			err := createZip(ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error creating zip file: %w", err)
				return
			}
		},
	}
)

var (
	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create config file, snippet template files, zip archive or the database schema",
		Long: `The create command is used to create the configuration file, snippet template files, or a zip archive of the database.

The configuration file is required to access any database, and will also allow you to set defaults for where you want snippets to be rendered. Run the following command to get started on a configuration file:

ask create conf

The snippet template files are examples of what snippets look like if you are just learning about ask.  There are two types of templates that can be generated, a regular text file to show the format of snippets using the following command:

ask create template

Additionally, you can create a text file and yaml file if you have metadata about snippets that you'd like to convert for use in ask.

ask create template --yaml

Lastly, you can create a zip archive of all the snippets in the configured ask database with the command

ask create zip
`,
		Args: cobra.MinimumNArgs(1),
	}
)

func createFullExample() (err error) {
	createTextExample()
	var snippetData structs.Snippet
	texFileContent := ""
	textFileName := "./example.txt"

	Variables := make(map[string]*structs.Variable)
	Variables["VAR1"] = &structs.Variable{Description: "VAR1 description goes here.", ExampleValue: "Example value of VAR1 goes here", DefaultValue: "Default value for VAR1 goes here"}
	yamlFile := "example.yaml"
	snippetData.Name = "example"
	snippetData.Version = "0.1.0"
	snippetData.Description = "Description of the snippet goes here"
	snippetData.Variables = Variables
	snippetData.SnippetText = texFileContent
	snippetData.SnippetFile = textFileName
	snippetYamlHandle := writeYaml(snippetData, yamlFile)
	snippetYamlHandle.Close()
	return nil
}
func createTextExample() (err error) {
	textFileName := "example.txt"
	textFileContent := `This is an example snippet.  Variables are indicated with the following rules: 
	1. Use double curly braces and a space to surround the variable name.
	2. Only letters and underscores are permitted in the variable name.

This is an example of a variable: {{ VAR1 }}

Best practice is to use all capital letters to make it stand out, but you do you.

It is also very helpful to provide lists of other snippets to do based on how this one worked
	#next snippet
	arp_scan
	ping_sweep`

	os.WriteFile(textFileName, []byte(textFileContent), 0644)
	fmt.Fprintf(os.Stdout, "[+] Wrote example.txt with example snippet.\n")
	return nil
}

func createDbSchema(db *sql.DB) (err error) {
	// Create the table schema in the database
	var schema string
	debug.Print("[*] Entered the createDbSchema function")
	debug.Print("[*] Checking for the snippets table")
	if ask_db.TableExists(db, "snippets") == false {
		debug.Print("Creating snippets table in database")
		if ask_db.Driver == "mysql" {
			debug.Print("[!] Assuming mysql driver")
			schema = `
	CREATE TABLE snippets (
		id INT(11) NOT NULL AUTO_INCREMENT,
		name TEXT DEFAULT NULL,
		description TEXT DEFAULT NULL,
		variables TEXT DEFAULT NULL,
		version TEXT DEFAULT NULL,
		snippetText TEXT DEFAULT NULL,
		PRIMARY KEY (id)
	);`
		} else {
			debug.Print("[!] Assuming sqlite3 driver for snippets table creation")
			schema = `
			CREATE TABLE IF NOT EXISTS snippets (
			    id INTEGER PRIMARY KEY AUTOINCREMENT,
			    name TEXT,
			    description TEXT,
			    variables TEXT,
			    version TEXT,
			    snippetText TEXT
			);`
		}

		_, err := db.Exec(schema)
		if err != nil {
			return fmt.Errorf("Failed to create schema: %v", err)
		}

		fmt.Println("[+] Created snippets table")

	}

	/*
			debug.Print("Checking for the snippets tagMap table")
			if ask_db.TableExists(db, "tagMap") == false {
				schema := `
		CREATE TABLE tagMap (
			id INT(11) NOT NULL AUTO_INCREMENT,
			tagId INT(11) DEFAULT NULL,
			snipId INT(11) DEFAULT NULL,
			PRIMARY KEY (id)
		);`

				_, err = db.Exec(schema)
				if err != nil {
					log.Fatalf("Failed to create schema: %v", err)
				}

				fmt.Println("Created tagMap table")
			}

			debug.Print("Checking for the snippets tags")
			if ask_db.TableExists(db, "tags") == false {
				schema := `
		CREATE TABLE tags (
			id INT(11) NOT NULL AUTO_INCREMENT,
			tag TEXT DEFAULT NULL,
			PRIMARY KEY (id)
		);`

				_, err = db.Exec(schema)
				if err != nil {
					log.Fatalf("Failed to create schema: %v", err)
				}

				fmt.Println("Created tags table")

				//create tagMap table
			}

	*/
	return nil
}

var (
	databaseCmd = &cobra.Command{
		Use:   "db_schema",
		Short: "Create the ask database schema on the server.",
		Aliases: []string{"db","schema"},
		Long: `This command will create the schemas to support ask.  Under the hood, it creates three tables:

- snippets ( id,name, description, variables, version, snippetText)
- tagMap ( id, tagId, snipId)
- tags ( id, tag)
		`,
		Args: cobra.ExactArgs(0),
		Run: func(_ *cobra.Command, _ []string) {
			err := createDbSchema(ask_db.DB)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Encountered an error generating the database schema: \n%w\n", err)
			}
			fmt.Fprintf(os.Stdout, "[+] Schema is created and ready to go")
		},
	}
)

var (
	textTemplateCmd = &cobra.Command{
		Use:   "template",
		Short: "Create an example snippet file",
		Long: `Create an example snippet as either a plain text file, or a yaml file for more advanced users to see how to ingest existing snippets.

EXAMPLES
ask create template
	#Creates a simple text file to model for ingestion

ask create template --yaml
	#Create a yaml file with metadata and a snippet file.  Do this only if you are comfortable with the yaml schema and already have some metadata about snippets and snippet variables. Note that the text file contains the snippet text while the yaml files contains the metadata and the path to the text file.`,

		Args: cobra.ExactArgs(0),
		Run: func(_ *cobra.Command, _ []string) {
			if Yaml == true {
				err := createFullExample()
				if err != nil {
					fmt.Fprintf(os.Stdout, "[!] Encountered an error generating a text file: %w\n", err)
				}
				fmt.Fprintf(os.Stdout, "[+] Edit your snippets and yaml file, and then run the following command to ingest it:\n\n ask add example.yaml\n")

				// put in the yaml creation
				return
			}
			err := createTextExample()
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Encountered an error generating a text file: %w\n", err)
			}

			fmt.Fprintf(os.Stdout, "[+] Edit your snippet, and then run the following command to ingest it:\n\n ask add example.txt\n")
		},
	}
)

func init() {
	createCmd.AddCommand(confCmd)
	createCmd.AddCommand(textTemplateCmd)
	textTemplateCmd.PersistentFlags().BoolVarP(&Yaml, "yaml", "y", false, "Create yaml file with metadata")
	createCmd.AddCommand(databaseCmd)
	// add flag to specify archive name
	createCmd.AddCommand(archiveCmd)
	databaseCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/ask/config.yaml)")
	databaseCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		// debug should be global, no?
		debug.Enabled = debugFlag
		if err := config.Init(cfgFile); err != nil {
			return err
		}
		_, err := ask_db.Init()
		return err
	}
	archiveCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/ask/config.yaml)")
	archiveCmd.PersistentFlags().StringVarP(&ArchiveFile, "outfile", "o", "archive.zip", "Zip file to create from database")
	archiveCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		// debug should be global, no?
		debug.Enabled = debugFlag
		if err := config.Init(cfgFile); err != nil {
			return err
		}
		_, err := ask_db.Init()
		return err
	}

}
