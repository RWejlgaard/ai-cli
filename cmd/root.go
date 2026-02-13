/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var apiKey string
var single string
var systemPrompt string
var model string
var verbose bool
var noColor bool
var quiet bool
var client anthropic.Client

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ai",
	Short: "A command-line tool for interacting with Anthropic's Claude model.",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		chat := Chat{
			client:       client,
			model:        model,
			systemPrompt: systemPrompt,
			verbose:      verbose,
			quiet:        quiet,
			noColor:      noColor,
		}
		if single != "" {
			chat.oneOff(single)
		} else {
			chat.startSession()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("api-key", "k", "", "Anthropic API key")
	viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))

	rootCmd.PersistentFlags().StringVarP(&single, "single", "s", "", "Prompt for a single response")

	rootCmd.PersistentFlags().StringP("system-prompt", "p", "", "Override the system prompt")
	viper.BindPFlag("system-prompt", rootCmd.PersistentFlags().Lookup("system-prompt"))

	rootCmd.PersistentFlags().StringP("model", "m", "", "Model to use")
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))

	rootCmd.PersistentFlags().BoolP("no-color", "n", false, "Disable color output")
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))

	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Disable prompts and system messages")
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

}

func initConfig() {
	viper.SetConfigName("ai-cli")         // name of config file (without extension)
	viper.SetConfigType("yaml")           // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME/.config/") // path to look for the config file in

	viper.SetDefault("api-key", "$ANTHROPIC_API_KEY")
	viper.SetDefault("system-prompt", "You're a helpful AI assistant.")
	viper.SetDefault("model", "claude-opus-4-6")
	viper.SetDefault("no-color", false)
	viper.SetDefault("quiet", false)
	viper.SetDefault("verbose", false)

	err := viper.ReadInConfig()
	if err != nil {
		viper.SafeWriteConfig()
	}

	err = viper.ReadInConfig()
	if err == nil {
		apiKey = viper.GetString("api-key")
		systemPrompt = viper.GetString("system-prompt")
		model = viper.GetString("model")
		noColor = viper.GetBool("no-color")
		quiet = viper.GetBool("quiet")
		verbose = viper.GetBool("verbose")
	} else {
		viper.SafeWriteConfig()
	}

	if apiKey == "$ANTHROPIC_API_KEY" || apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	// Initialize the Anthropic client
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	client = anthropic.NewClient(opts...)

}
