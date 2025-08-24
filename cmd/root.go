package cmd

import (
	"ask/ask_db"
	"ask/config"
	"ask/internal/debug"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/fang"
	//We might use the mysql or sqlite3 driver based on user configs
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile   string
	debugFlag bool
	db        *sql.DB
)

// NewRootCommand is used to combine viper and cobra
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ask [subcommand]",
		Short: "An Aggregated Snippet Knowledgebase written in Golang",
		Long: `Ask (Aggregated Snippet Knowledgebase) by the OtherDave (https://github.com/clevernamestaken) is a CLI tool written in golang to save, edit, and share code snippets for terminal-based collaborative operations using databases and structured TUI forms.

EXAMPLES
ask create config
	#Create the config file to start interacting with a database or create a new one

ask create template
	#Create an example snippet file

ask add example.txt
	#Ingest the snippet file into the database

ask cat example
	#Look at the raw snippet text for the snippet named "example"

ask edit example
	#Edit a snippet called "example"

ask ls
	#Examine which snippets are in the database

ask ls mple
	#Examine which snippets in the database have a name containing the string "mple"

ask render example
	#Render the snippet named "example" to stdout after being prompted to fill out the variables

ask render text --all --outdir ./text_snippets
	#Dump the entire database in text format to a directory called ./text_snippets

ask render create zip --outfile archive.zip
	#Create an archive of the database

ask rm --prune
	#Remove all of the outdated snippets from the database

ask  add archive.zip
	#Add an archived zip to an existing database.

ask console
	#Start a metasploit-like console to search for and use snippets
				`,
	}
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.config/ask/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "enable debug output")
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(renderCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(browseCmd)
	rootCmd.AddCommand(consoleCmd)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		debug.Enabled = debugFlag
		parts := strings.Split(cmd.CommandPath(), " ")
		if parts[1] == "create" {
			return nil
		}

		initializeConfig(cmd)
		_, err := ask_db.Init()
		if err != nil {
			return err
		}
		return nil

	}

	return rootCmd
}

func quickStartConfig(viper *viper.Viper) (v *viper.Viper, err error) {
	//If we do not have a config, prompt user to quick start or use the wizard
	debug.Print("[*] Entering quickStartConfig function because no config was found.")

	prompt := ""
promptLoop:
	for {
		fmt.Printf("[!] No configuration found.  Do you want to\n1)Go to config wizard to create one or\n2)Create a default config to get started.\n")
		fmt.Scanln(&prompt)
		switch prompt {
		case "1":
			viper, err := configWizard()
			if err != nil {
				return viper, err
			}
			break promptLoop
		case "2":
			viper, err := quickConfig()
			if err != nil {
				return viper, err
			}
			break promptLoop
		default:
			fmt.Println("[!] Invalid option. Please try again.")
		}
	}

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	viper.AddConfigPath(home + "/.config/ask")
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	viper.ReadInConfig()
	//connect to the database
	username := viper.GetString("username")
	password := viper.GetString("password")
	dbname := viper.GetString("database")
	connection := viper.GetString("port")
	var db *sql.DB

	// Create the database if it doesn't exist
	if connection != "" {
		// If mysql is listed for the connection type, use mysql
		ask_db.Driver = "mysql"
		debug.Print("[!] Assuming the sql driver is mysql")
		host := viper.GetString("host")
		port := viper.GetString("port")
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", username, password, host, port)
		debug.Print("[*] Attempting connection to mysql server: %s", dsn)
		db, err = sql.Open("mysql", dsn)

		debug.Print("[*] Attempting create database if it does not exist: %s%s", dsn, dbname)
		query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbname)
		_, err = db.Exec(query)
		if err != nil {
			fmt.Fprintf(os.Stdout, "[!] Failed to access database.  Check your configuration file and the database to ensure access.\n\n%v\n", err)
			os.Exit(1)
		}

		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, dbname)
		debug.Print("[*] Attempting connect to the database: %s", dsn)

		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("error opening DB: %w", err)
		}
	} else {
		//If mysql is not listed, let's assume it is a sql database
		debug.Print("[!] Using the sql driver is sqlite3")

		sqlFile := viper.GetString("sqlFile")

		if sqlFile[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			sqlFile = strings.Replace(sqlFile, "~", homeDir, 1)
		}

		debug.Print("[*] Attempting connection to sqlite: %s", sqlFile)

		db, err = sql.Open("sqlite3", sqlFile)
		if err != nil {
			log.Fatalf("[!] Failed to open database: %v", err)
		}
	}

	debug.Print("[+] Succesfully opened database")

	//create the schema

	createDbSchema(db)
	return viper, nil

}

func initializeConfig(cmd *cobra.Command) error {
	debug.Print("[*] Attempting to initialize config")
	v := viper.New()
	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		v.AddConfigPath(home + "/.config/ask")
		v.SetConfigType("yaml")
		v.SetConfigName("config")
	}

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err == nil {
		debug.Print("[*] Using config file: %s", v.ConfigFileUsed())
	} else {
		v, err = quickStartConfig(v)
		if err != nil {
			return err
		}
		v.ReadInConfig()
	}
	bindflags(cmd, v)

	if err := config.Init(cfgFile); err != nil {
		return err
	}

	return nil
}

func bindflags(cmd *cobra.Command, v *viper.Viper) {
	//Allow flags to override config file settings
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

// Execute helps combine viper and cobra
func Execute() (err error) {
	debug.Print("Executing the Execute function of root")
	cmd := NewRootCommand()
	if err = fang.Execute(context.Background(), cmd); err != nil {
		os.Exit(1)
	}

	return err
}
