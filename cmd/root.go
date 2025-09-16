package cmd

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unchromed_launcher/constants"
	"unchromed_launcher/globals"
	"unchromed_launcher/logger"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var logfile os.File

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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", constants.EMPTY, "config file (default is $HOME/.unchromed_launcher.yaml)")

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

		// Add the Unchromed Launcher configuration file and extension type.
		viper.SetConfigType("yaml")
		viper.SetConfigName(".unchromed_launcher")
	}

	// Read in environment variables that match.
	viper.AutomaticEnv()

	// Set various defaults.
	viper.SetDefault(constants.DEBUG, false)
	viper.SetDefault(constants.PAUSE_AFTER_RUN, false)
	viper.SetDefault(constants.PAUSE_ON_UPDATE, false)
	viper.SetDefault(constants.BIN_DIRECTORY, filepath.Join(".", "bin"))
	viper.SetDefault(constants.DOWNLOAD_DIRECTORY, filepath.Join(".", "download"))
	viper.SetDefault(constants.PROFILE_DIRECTORY, filepath.Join(".", "profile"))
	viper.SetDefault(constants.INSTALLED_VERSION, constants.EMPTY)
	viper.SetDefault(constants.CHROME_DISTRIBUTION, constants.UNCHROMED_CHROMIUM_DISTRIBUTION)
	viper.SetDefault(constants.CHROME_COMMAND_LINE_OPTIONS, "--no-default-browser-check")

	// Read the configuration file.
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// No config file, just use defaults.
			viper.SafeWriteConfig()
			viper.ReadInConfig()
			log.Printf("%s: Unable to load config file, writing default values to [%s].\n\n",
				color.HiBlueString(constants.INFO_NORMAL_CASE), viper.ConfigFileUsed())
		} else {
			log.Fatalf("%s: Error reading config file: %s\n",
				color.RedString(constants.FATAL_NORMAL_CASE), err.Error())
			os.Exit(1)
		}
	}

	// If debugging is set, log everything to the log file.
	if viper.GetBool(constants.DEBUG) {
		logger.EnableFileLogging()
	}

	// Make sure the CHROME_DISTRIBUTION is set to one of our supported distributions.
	distribution := viper.GetString(constants.CHROME_DISTRIBUTION)
	if strings.EqualFold(distribution, constants.UNCHROMED_CHROMIUM_DISTRIBUTION) == false &&
		strings.EqualFold(distribution, constants.UNCHROMED_WINCHROME_DISTRIBUTION) == false &&
		strings.EqualFold(distribution, constants.CROMITE_DISTRIBUTION) == false {
			log.Fatalf("%s: Unsupported distribution[%s] found. Valid distributions are '%s', '%s',and '%s'.\n",
				color.RedString(constants.FATAL_NORMAL_CASE), distribution,
				constants.UNCHROMED_CHROMIUM_DISTRIBUTION, constants.UNCHROMED_WINCHROME_DISTRIBUTION,
				constants.CROMITE_DISTRIBUTION)
			os.Exit(1)
		}

	// Use the global ExeDir to make sure the necessary directories exist. If
	// they do not exist, they are created.
	if viper.GetBool(constants.DEBUG) {
    	log.Println("Executable directory:", globals.ExeDir)
	}

	// Make sure the DOWNLOAD_DIRECTORY exists.
    downloadDirectory := filepath.Join(globals.ExeDir, viper.GetString(constants.DOWNLOAD_DIRECTORY))
	_, err = os.Stat(downloadDirectory)
	if os.IsNotExist(err) {
		err := os.MkdirAll(downloadDirectory, 0755)
		if err != nil {
			log.Fatalf("%s: Failed to create download directory[%s]. Error[%s]\n",
				color.RedString(constants.FATAL_NORMAL_CASE), downloadDirectory, err.Error())
			os.Exit(1)
		}
	}

	// Make sure the PROFILE_DIRECTORY exists.
    profileDirectory := filepath.Join(globals.ExeDir, viper.GetString(constants.PROFILE_DIRECTORY))
	_, err = os.Stat(profileDirectory)
	if os.IsNotExist(err) {
		err := os.MkdirAll(profileDirectory, 0755)
		if err != nil {
			log.Fatalf("%s: Failed to create profile directory[%s]. Error[%s]\n",
				color.RedString(constants.FATAL_NORMAL_CASE), profileDirectory, err.Error())
			os.Exit(1)
		}
	}

	// Make sure the BIN_DIRECTORY exists.
    binDirectory := filepath.Join(globals.ExeDir, viper.GetString(constants.BIN_DIRECTORY))
	_, err = os.Stat(binDirectory)
	if os.IsNotExist(err) {
		err := os.MkdirAll(binDirectory, 0755)
		if err != nil {
			log.Fatalf("%s: Failed to create bin directory[%s]. Error[%s]\n",
				color.RedString(constants.FATAL_NORMAL_CASE), binDirectory, err.Error())
			os.Exit(1)
		}
	}
}
