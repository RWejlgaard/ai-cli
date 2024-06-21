/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)


var apiKey string
var single string
var systemPrompt string
var model string
var verbose bool
var client *openai.Client

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ai",
	Short: "A command-line tool for interacting with OpenAI's GPT-4o model.",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) { 
		if single != "" {
			oneOff(client, model, systemPrompt, single, verbose)
		} else {
			startSession(client, model, systemPrompt, verbose)
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ai.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	
	// Option for setting the API key
	rootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "k", "$OPENAI_API_KEY", "OpenAI API key")
	rootCmd.PersistentFlags().StringVarP(&single, "single", "s", "", "Prompt for a single response")
	rootCmd.PersistentFlags().StringVarP(&systemPrompt, "system-prompt", "p", "", "Override the system prompt")
	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "gpt-4o", "Model to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	if apiKey == "$OPENAI_API_KEY" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	// Initialize the OpenAI client
	client = openai.NewClient(apiKey)

}


