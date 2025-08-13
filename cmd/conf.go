package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ask/internal/debug"
	structs "ask/internal/model"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func quickConfig() (v *viper.Viper, err error) {
	//create default config file
	debug.Print("[*] Entered quickConfig function.")
	configName := "config.yaml"
	yamlContent := `#This yaml file was created through a quick config.  Edit at your own risk or use ask create config to create a new config file

sqlFile: ~/.config/ask/ask.db
ingest_dir: /tmp/raw_snippets
outputdir: ~/.config/ask/snippets`

	home, err := os.UserHomeDir()
	OutputDir := home + "/.config/ask"
	err = os.MkdirAll(OutputDir, os.ModePerm)
	if err != nil {
		return v, fmt.Errorf("failed to create directory: %w", err)
	}
	debug.Print("Directory created: %s", OutputDir)
	os.WriteFile(OutputDir+"/"+configName, []byte(yamlContent), 0644)
	debug.Print("[+] Created quickConfig.")

	sqlFile := OutputDir + "/ask.db"
	_, err = os.Create(sqlFile)
	if err != nil {
		log.Fatal(err)
	}

	v = viper.New()
	v.AddConfigPath(home + "/.config/ask")
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AutomaticEnv()

	return v, nil
}
func configWizard() (v *viper.Viper, err error) {
	Username := "root"
	var Password string
	var dbType string
	Database := "ask"
	Host := "127.0.0.1"
	Port := "3306"
	OutputDir := "~/.config/nvim/UltiSnips/all"
	sqlFile := "~/.config/ask/ask.db"
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return v, err
	}

	configPath := filepath.Join(homeDir, "/.config/ask/config.yaml")
	_, err = os.Stat(configPath)
	if err == nil {

	prompt := ""
promptLoop:
	for {
		fmt.Printf("[!] Found a configuration file at ~/.config/ask/config.yaml.  Do you want to overwrite it?\nY/N")
		fmt.Scanln(&prompt)
		switch prompt {
		case "Y":
			os.Remove(configPath)
			break promptLoop
		case "N":
			os.Exit(1)
		default:
			fmt.Println("[!] Invalid option. Please try again.")
		}
	}

	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which type of database do you want?").
				Options(
					huh.NewOption("Local", "local"),
					huh.NewOption("Remote", "remote"),
				).
				Value(&dbType),
		),
	)

	// Run the form and process result
	if err := form.Run(); err != nil {
		panic(err)
	}

	if dbType == "remote" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Username to connect to mysql database").
					Value(&Username),
				huh.NewInput().
					Title("Password to connect to mysql database").
					Value(&Password),
				huh.NewInput().
					Title("Database").
					Value(&Database),
				huh.NewInput().
					Title("Host").
					Value(&Host),
				huh.NewInput().
					Title("Port").
					Value(&Port).
					Validate(func(str string) error {
						port, err := strconv.Atoi(str)
						if err != nil {
							return errors.New("Port must be a valid number")
						}
						if port < 1 || port > 65535 {
							return errors.New("Port must be between 1 and 65535")
						}
						return nil
					}),
				huh.NewInput().
					Title("OutputDir").
					Description("This is where snippets where be saved when you use the `render` command").
					Value(&OutputDir),
			),
		)

		err = form.Run()
		if err != nil {
			return v, err
		}

		// write to yaml file
		fmt.Fprintf(os.Stdout, "[+] Writing config yaml file to $HOME/.config/ask/config.yaml\n")
		newConfig := structs.Config{
			Username:  Username,
			Password:  Password,
			Database:  Database,
			Host:      Host,
			Port:      Port,
			OutputDir: OutputDir,
		}


		os.MkdirAll(homeDir+"/.config/ask", os.ModePerm)
		yamlHandle, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return v, err
		}

		yaml.NewEncoder(yamlHandle).Encode(newConfig)
		yamlHandle.Close()
	} else if dbType == "local" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Path to sqlite database").
					Value(&sqlFile),
				huh.NewInput().
					Title("OutputDir").
					Description("This is where snippets where be saved when you use the `render` command").
					Value(&OutputDir),
			),
		)
		err = form.Run()
		if err != nil {
			return v, err
		}

		// write to yaml file
		fmt.Fprintf(os.Stdout, "[+] Writing config yaml file to $HOME/.config/ask/config.yaml\n")
		newConfig := structs.ConfigLocal{
			SQLFile:   sqlFile,
			OutputDir: OutputDir,
		}

		configPath := filepath.Join(homeDir, "/.config/ask/config.yaml")

		os.MkdirAll(homeDir+"/.config/ask", os.ModePerm)
		yamlHandle, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return v, err
		}

		yaml.NewEncoder(yamlHandle).Encode(newConfig)
		yamlHandle.Close()

		if sqlFile[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			sqlFile = strings.Replace(sqlFile, "~", homeDir, 1)
		}

		// If it doesn't exist, create it
		_, err = os.Create(sqlFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	v = viper.New()
	v.AddConfigPath(homeDir + "/.config/ask")
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AutomaticEnv()
	return v, nil
}

var (
	confCmd = &cobra.Command{
		Use:   "conf",
		Short: "Create config file",
		Aliases: []string{"config"},
		Long: `Generate an ask configuration file for connection to MySql database and where to save files to

The configuration file will include information related to the database to store and query snippet information.

An example configuration file might look like this:


username: root
password: your_secure_password
database: ask
host: 127.0.0.1
port: "3306"
outputdir: ~/.config/nvim/UltiSnips/all
		`,
		Args: cobra.MinimumNArgs(0),
		Run: func(_ *cobra.Command, _ []string) {
			_, err := configWizard()
			if err != nil {
				fmt.Fprintf(os.Stdout, "[!] Error generating config: %w", err)
			}
		},
	}
)
