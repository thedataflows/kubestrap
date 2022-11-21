/*
Copyright © 2022 Dataflows

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"dataflows.com/kubestrap/internal/pkg/files"
	"dataflows.com/kubestrap/internal/pkg/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultConfigPath     = "./"
	defaultConfigFileName = "kubestrap.yaml"
	viperEnvPrefix        = "KS"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubestrap",
	Short: "Toolbox for easy bootstrap of self service kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Long = cmd.Short + fmt.Sprintf("\n\nAll flags values can be provided via env vars starting with %s_*\nTo pass a subcommand (e.g. 'flux') flag, use %s_FLUX_FLAGNAME=somevalue", viperEnvPrefix, viperEnvPrefix)
		cmd.Help()
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	flags := pflag.NewFlagSet("root", pflag.PanicOnError)
	flags.StringP("log-level", "l", logging.InfoLevel.String(), fmt.Sprintf("Set log level to one of: %s", logging.LogLevelsStr))
	flags.StringVar(&cfgFile, "config", defaultConfigPath+defaultConfigFileName, "Config file override")

	rootCmd.PersistentFlags().AddFlagSet(flags)
	viper.BindPFlags(flags)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(defaultConfigPath)
		configPath, err := files.AppHome("")
		logging.ExitOnError(err, 1)
		viper.AddConfigPath(configPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName(files.TrimExtension(defaultConfigFileName))
	}

	viper.SetEnvPrefix(viperEnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		logging.Logger.Warnf("%s", err)
	}

	logging.Logger.SetLevel(logging.ParseLevel(viper.GetString("log-level")))
}

// CheckRequiredFlags exits with error when one ore more required flags are not set
func CheckRequiredFlags(prefixKey string, requiredFlags []string, cmd *cobra.Command) {
	unsetFlags := make([]string, 0, len(requiredFlags))
	for _, f := range requiredFlags {
		if !viper.GetViper().IsSet(prefixKey + f) {
			unsetFlags = append(unsetFlags, f)
		}
	}
	if len(unsetFlags) > 0 {
		fmt.Fprintln(os.Stderr, "Error: required flags are not set:")
		for _, f := range unsetFlags {
			fmt.Fprintf(os.Stderr, "  --%s", f)
		}
		fmt.Fprintf(os.Stderr, "\n\n")
		cmd.Usage()
		os.Exit(1)
	}
}
