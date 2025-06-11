package cmd

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"ungoogled_launcher/constants"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   constants.APPLICATION_NAME_LOWERCASE,
	Short: constants.ROOT_SHORT_DESCRIPTION,
	Long:  constants.ROOT_LONG_DESCRIPTION,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(defaultCommand string) {
	var commandFound bool
  	cmd := rootCmd.Commands()

  	for _, a := range cmd {
    	for _, b := range os.Args[1:] {
      		if a.Name() == b {
       			commandFound = true
        		break
      		}
    	}
  	}

  	if commandFound == false {
    	args := append([]string{defaultCommand}, os.Args[1:]...)
    	rootCmd.SetArgs(args)
  	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%s: %s\n", color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
    	os.Exit(1)
  	}
}

func init() {
	cobra.MousetrapHelpText = constants.EMPTY
	cobra.OnInitialize(initConfig)

	cobra.AddTemplateFunc("StyleHeading", color.New(color.FgGreen).SprintFunc())
	usageTemplate := rootCmd.UsageTemplate()
	usageTemplate = strings.NewReplacer(
		`Usage:`, `{{StyleHeading "Usage:"}}`,
		`Aliases:`, `{{StyleHeading "Aliases:"}}`,
		`Available Commands:`, `{{StyleHeading "Available Commands:"}}`,
		`Global Flags:`, `{{StyleHeading "Global Flags:"}}`,
		// The following one steps on "Global Flags:"
		`Flags:`, `{{StyleHeading "Flags:"}}`,
	).Replace(usageTemplate)
	re := regexp.MustCompile(`(?m)^Flags:\s*$`)
	usageTemplate = re.ReplaceAllLiteralString(usageTemplate, `{{StyleHeading "Flags:"}}`)
	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.SetOut(color.Output)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", constants.EMPTY, "config file (default is $HOME/.ungoogled_launcher.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().Bool("help", false, constants.HELP_SHORT_DESCRIPTION)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Find home directory.
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	if cfgFile != constants.EMPTY {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Add the XDG_CONFIG_HOME directory to the search path, if configured.
		xdgConfigHome, found := os.LookupEnv("XDG_CONFIG_HOME")
		if found {
			xdgConfigPath := filepath.Join(xdgConfigHome, constants.APPLICATION_NAME_LOWERCASE)
			_, err := os.Stat(xdgConfigPath)
			if os.IsNotExist(err) {
				// Do nothing.
			} else {
				viper.AddConfigPath(xdgConfigPath)
			}
		}

		// Add the user's home directory to the search path.
		viper.AddConfigPath(home)

		// Add the Ungoogled Launcher configuration file and extension type.
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ungoogled_launcher")
	}

	// Read in environment variables that match.
	viper.AutomaticEnv()

	// Set various defaults.
	viper.SetDefault(constants.DEBUG, false)
	viper.SetDefault(constants.BIN_DIRECTORY, filepath.Join(".", "bin"))
	viper.SetDefault(constants.DOWNLOAD_DIRECTORY, filepath.Join(".", "download"))
	viper.SetDefault(constants.PROFILE_DIRECTORY, filepath.Join(".", "profile"))
	viper.SetDefault(constants.INSTALLED_VERSION, constants.EMPTY)
	viper.SetDefault(constants.CHROME_COMMAND_LINE_OPTIONS, "--no-default-browser-check")

	// Make sure the DOWNLOAD_DIRECTORY exists.
	_, err = os.Stat(viper.GetString(constants.DOWNLOAD_DIRECTORY))
	if os.IsNotExist(err) {
		err := os.MkdirAll(viper.GetString(constants.DOWNLOAD_DIRECTORY), 0755)
		if err != nil {
			log.Fatalf("%s: Failed to create download directory[%s]. Error[%s]\n",
				color.RedString(constants.FATAL_NORMAL_CASE), viper.GetString(constants.DOWNLOAD_DIRECTORY), err.Error())
			os.Exit(1)
		}
	}

	// Make sure the PROFILE_DIRECTORY exists.
	_, err = os.Stat(viper.GetString(constants.PROFILE_DIRECTORY))
	if os.IsNotExist(err) {
		err := os.MkdirAll(viper.GetString(constants.PROFILE_DIRECTORY), 0755)
		if err != nil {
			log.Fatalf("%s: Failed to create profile directory[%s]. Error[%s]\n",
				color.RedString(constants.FATAL_NORMAL_CASE), viper.GetString(constants.PROFILE_DIRECTORY), err.Error())
			os.Exit(1)
		}
	}

	// Make sure the BIN_DIRECTORY exists.
	_, err = os.Stat(viper.GetString(constants.BIN_DIRECTORY))
	if os.IsNotExist(err) {
		err := os.MkdirAll(viper.GetString(constants.BIN_DIRECTORY), 0755)
		if err != nil {
			log.Fatalf("%s: Failed to create bin directory[%s]. Error[%s]\n",
				color.RedString(constants.FATAL_NORMAL_CASE), viper.GetString(constants.BIN_DIRECTORY), err.Error())
			os.Exit(1)
		}
	}

	// Read the configuration file.
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// No config file, just use defaults.
			viper.SafeWriteConfig()
			log.Printf("%s: Unable to load config file, using/writing default values to [%s].\n\n",
				color.HiBlueString(constants.INFO_NORMAL_CASE), viper.ConfigFileUsed())
		} else {
			log.Fatalf("%s: Error reading config file: %s\n",
				color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
			os.Exit(1)
		}
	}

	if viper.GetBool(constants.DEBUG) {
	}
}
