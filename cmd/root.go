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
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dataflows.com/kubestrap/internal/pkg/files"
	"dataflows.com/kubestrap/internal/pkg/kubestrap"
	"dataflows.com/kubestrap/internal/pkg/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	viperEnvPrefix  = "KS"
	viperConfigType = "yaml"
)

var (
	userConfigPaths    []string
	defaultConfigPaths = []string{"."}
	kubernetesConfig   = files.GetKubeconfigPath()
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

	programPath, err := files.CurrentProcessPath()
	logging.ExitOnError(err, 1)
	defaultConfigName := files.TrimExtension(filepath.Base(programPath))
	viper.SetConfigName(defaultConfigName)

	configPath, err := files.AppHome("")
	logging.ExitOnError(err, 1)
	defaultConfigPaths = append(defaultConfigPaths, configPath)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	flags := pflag.NewFlagSet("root", pflag.PanicOnError)
	flags.StringP("log-level", "l", logging.InfoLevel.String(), fmt.Sprintf("Set log level to one of: %s", logging.LogLevelsStr))
	flags.StringArrayVar(
		&userConfigPaths, "config", defaultConfigPaths, fmt.Sprintf(
			"Config file(s) or directories. When just dirs, file '%s' with extensions '%s' is looked up. Can be specified multiple times",
			defaultConfigName,
			strings.Join(viper.SupportedExts, ", "),
		),
	)

	rootCmd.PersistentFlags().AddFlagSet(flags)
	viper.BindPFlags(flags)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix(viperEnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// viper.SetConfigType(viperConfigType)

	// Use config file from the flag.
	for _, p := range userConfigPaths {
		if files.IsFile(p) {
			viper.SetConfigName(files.TrimExtension(filepath.Base(p)))
			p = filepath.Dir(p)
		}
		viper.AddConfigPath(p)
		if err := viper.MergeInConfig(); err != nil {
			logging.Logger.Warnf("%s", err)
		}
	}

	logging.Logger.SetLevel(logging.ParseLevel(viper.GetString("log-level")))
	if logging.Logger.Level == logging.TraceLevel {
		logging.Logger.Debugln("====== begin viper configuration dump ======")
		viper.DebugTo(logging.Logger.WriterLevel(logging.Logger.Level))
		time.Sleep(100 * time.Millisecond)
		logging.Logger.Debugln("====== end viper configuration dump ======")
	}
}

// checkRequiredFlags exits with error when one ore more required flags are not set
func checkRequiredFlags(cmd *cobra.Command, requiredFlags []string) {
	unsetFlags := make([]string, 0, len(requiredFlags))
	for _, f := range requiredFlags {
		if !viper.GetViper().IsSet(prefixKey(cmd, f)) {
			unsetFlags = append(unsetFlags, f)
		}
	}
	if len(unsetFlags) > 0 {
		fmt.Fprintln(os.Stderr, "Error: required flags are not set:")
		for _, f := range unsetFlags {
			fmt.Fprintf(os.Stderr, "  --%s\n", f)
		}
		fmt.Fprintf(os.Stderr, "\n")
		cmd.Usage()
		os.Exit(1)
	}
}

// prefixKey prepends current and parent Use to specified key name
func prefixKey(cmd *cobra.Command, keyName string) string {
	parentKey := ""
	for cmd != nil && cmd != cmd.Root() {
		parentKey = kubestrap.ConcatStrings(cmd.Use, ".", parentKey)
		cmd = cmd.Parent()
	}
	if keyName == "" && parentKey[len(parentKey)-1:] == "." {
		return parentKey[:len(parentKey)-1]
	}
	return parentKey + keyName
}

// appendStringArgsf appends viper value to existing args slice with optional formatted output with key and value
func appendStringArgsf(cmd *cobra.Command, args []string, key string, format string) []string {
	val := viper.GetViper().GetString(prefixKey(cmd, key))
	if val != "" {
		args = append(args, fmt.Sprintf(format, key, val))
	}
	return args
}

// appendStringArgs appends viper value to existing args slice
func appendStringArgs(cmd *cobra.Command, args []string, key string) []string {
	return appendStringArgsf(cmd, args, key, "")
}

// appendSplitArgs appends viper value to existing args slice after splitting them by splitPattern (default regex whitespace)
func appendStringSplitArgs(cmd *cobra.Command, args []string, key string, splitPattern string) []string {
	if splitPattern == "" {
		splitPattern = `\s+`
	}
	val := viper.GetViper().GetString(prefixKey(cmd, key))
	if val != "" {
		args = append(args, regexp.MustCompile(splitPattern).Split(val, -1)...)
	}
	return args
}

// viperBindPFlag is a convenience wrapper over viper.BindPFlag for local flags
func viperBindPFlag(cmd *cobra.Command, name string) {
	viper.BindPFlag(prefixKey(cmd, name), cmd.Flags().Lookup(name))
}

// viperBindPersistentPFlag is a convenience wrapper over viper.BindPFlag for persistent flags
func viperBindPersistentPFlag(cmd *cobra.Command, name string) {
	viper.BindPFlag(prefixKey(cmd, name), cmd.PersistentFlags().Lookup(name))
}
