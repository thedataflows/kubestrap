package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/thedataflows/go-commons/pkg/defaults"
	"github.com/thedataflows/go-commons/pkg/file"
	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/process"
	"github.com/thedataflows/go-commons/pkg/stringutil"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Options struct {
	EnvPrefix       string
	ConfigType      string
	ConfigName      string
	UserConfigPaths []string
	LogLevelKey     string
	LogFormatKey    string
	Flags           *pflag.FlagSet
}

type Option func(*Options)

func WithEnvPrefix(envPrefix string) Option {
	return func(o *Options) {
		o.EnvPrefix = envPrefix
	}
}

func WithConfigType(configType string) Option {
	return func(o *Options) {
		o.ConfigType = configType
	}
}

func WithConfigName(configName string) Option {
	return func(o *Options) {
		o.ConfigName = configName
	}
}

func WithUserConfigPaths(userConfigPaths []string) Option {
	return func(o *Options) {
		o.UserConfigPaths = userConfigPaths
	}
}

func WithLogLevelKey(logLevelKey string) Option {
	return func(o *Options) {
		o.LogLevelKey = logLevelKey
	}
}

func WithLogFormatKey(logFormatKey string) Option {
	return func(o *Options) {
		o.LogFormatKey = logFormatKey
	}
}

func WithFlags(flags *pflag.FlagSet) Option {
	return func(o *Options) {
		o.Flags = flags
	}
}

// NewOptions sets default Options overriding with options
func NewOptions(options ...Option) *Options {
	opts := Options{
		EnvPrefix: defaults.ViperEnvPrefix,
	}
	opts.ConfigName = file.TrimExtension(filepath.Base(process.CurrentProcessPath()))
	configPath, err := file.AppHome("")
	if err != nil {
		log.Fatal(err)
	}
	opts.UserConfigPaths = []string{".", configPath}
	opts.LogLevelKey = defaults.LogLevelKey
	opts.LogFormatKey = defaults.LogFormatKey

	opts.Flags = pflag.NewFlagSet("root", pflag.ExitOnError)
	opts.Flags.String(
		opts.LogLevelKey,
		log.InfoLevel.String(),
		fmt.Sprintf("Set log level to one of: '%s'",
			strings.Join(log.AllLevelsValues, ", "),
		),
	)
	opts.Flags.String(opts.LogFormatKey, log.LogFormats[0], fmt.Sprintf("Set log format to one of: '%s'", strings.Join(log.LogFormats, ", ")))
	opts.Flags.StringSliceVar(
		&opts.UserConfigPaths,
		"config",
		opts.UserConfigPaths, fmt.Sprintf(
			"Config file(s) or directories. When just dirs, file '%s' with extensions '%s' is looked up. Can be specified multiple times",
			opts.ConfigName,
			strings.Join(viper.SupportedExts, ", "),
		),
	)

	// override defaults with options
	for _, o := range options {
		o(&opts)
	}

	return &opts
}

// InitConfig reads in config file and ENV variables if set.
func (opts *Options) InitConfig() {
	if opts == nil {
		opts = NewOptions()
	}

	viper.SetEnvPrefix(opts.EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	opts.setLogging()

	viper.SetConfigType(opts.ConfigType)
	// Use config file from the flag.
	for _, p := range opts.UserConfigPaths {
		viper.SetConfigName(opts.ConfigName)
		if !file.IsAccessible(p) {
			log.Warnf("'%s' is not accessible!", p)
			continue
		}
		if file.IsFile(p) {
			viper.SetConfigName(file.TrimExtension(filepath.Base(p)))
			p = filepath.Dir(p)
		}
		viper.AddConfigPath(p)
		if err := viper.MergeInConfig(); err != nil {
			log.Warnf("%s", err)
		}
	}

	// a second call is to set again logging if configured in file
	opts.setLogging()

	if log.Logger.GetLevel() == log.TraceLevel {
		log.Trace("====== begin viper configuration dump ======")
		viper.DebugTo(log.Logger)
		time.Sleep(100 * time.Millisecond)
		log.Trace("====== end viper configuration dump ======")
	}

	// TODO maybe enable WatchConfig() if finding a method to override the viper Logger with ours
	// Perhaps via the Options interface?
	// Option configures Viper using the functional options paradigm popularized by Rob Pike and Dave Cheney.
	// If you're unfamiliar with this style,
	// see https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html and
	// https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis.
	// viper.WatchConfig()
}

func (opts *Options) setLogging() {
	// Set log format
	v := viper.GetString(opts.LogFormatKey)
	if len(v) == 0 {
		v = log.LogFormats[0]
	}
	err := log.SetLogFormat(v)
	if err != nil {
		log.Fatal(err)
	}

	// Set log level
	v = viper.GetString(opts.LogLevelKey)
	if len(v) == 0 {
		v = log.InfoLevel.String()
	}
	err = log.SetLogLevel(v)
	if err != nil {
		log.Fatal(err)
	}
}

// CheckRequiredFlags exits with error when one ore more required flags are not set
func CheckRequiredFlags(cmd *cobra.Command, requiredFlags []string, ret int) {
	neededFlags := make([]string, 0, len(requiredFlags))
	for _, f := range requiredFlags {
		if !ViperIsSet(cmd, f) {
			neededFlags = append(neededFlags, f)
		}
	}
	if len(neededFlags) > 0 {
		log.Error("Error: required flags are not set:")
		for _, f := range neededFlags {
			log.Errorf("  --%s", f)
		}
		_ = cmd.Usage()
		if ret > 0 {
			os.Exit(ret)
		}
	}
}

// BuildEnvKey returns a fully constructed environment variable name
func BuildEnvKey(cmd *cobra.Command, envPrefix string, keyName string) string {
	if len(envPrefix) == 0 {
		envPrefix = defaults.ViperEnvPrefix
	}
	parentKey := stringutil.ConcatStrings(envPrefix, "_")

	for cmd != nil && cmd != cmd.Root() {
		parentKey = stringutil.ConcatStrings(cmd.Use, "_", parentKey)
		cmd = cmd.Parent()
	}
	if keyName == "" && parentKey[len(parentKey)-1:] == "_" {
		return parentKey[:len(parentKey)-1]
	}
	return strings.ToUpper(parentKey + keyName)
}

// PrefixKey prepends current and parent Use to specified key name
func PrefixKey(cmd *cobra.Command, keyName string) string {
	parentKey := ""
	for cmd != nil && cmd != cmd.Root() {
		parentKey = stringutil.ConcatStrings(cmd.Use, ".", parentKey)
		cmd = cmd.Parent()
	}
	if keyName == "" && parentKey[len(parentKey)-1:] == "." {
		return parentKey[:len(parentKey)-1]
	}
	return parentKey + keyName
}

// AppendStringArgsf appends viper value to existing args slice with optional formatted output with key and value
func AppendStringArgsf(format string, cmd *cobra.Command, args []string, key string) []string {
	val := ViperGetString(cmd, key)
	if val != "" {
		args = append(args, fmt.Sprintf(format, key, val))
	}
	return args
}

// AppendStringArgs appends viper value to existing args slice
func AppendStringArgs(cmd *cobra.Command, args []string, key string) []string {
	return AppendStringArgsf("", cmd, args, key)
}

// AppendSplitArgs appends viper value to existing args slice after splitting them by splitPattern (default regex whitespace)
func AppendStringSplitArgs(cmd *cobra.Command, args []string, key string, splitPattern string) []string {
	if splitPattern == "" {
		splitPattern = `\s+`
	}
	val := ViperGetString(cmd, key)
	if val != "" {
		args = append(args, regexp.MustCompile(splitPattern).Split(val, -1)...)
	}
	return args
}

// ViperBindPFlag is a convenience wrapper over viper.BindPFlag for local flags
func ViperBindPFlag(cmd *cobra.Command, name string) {
	_ = viper.BindPFlag(PrefixKey(cmd, name), cmd.Flags().Lookup(name))
}

// ViperBindPFlagSet is a convenience wrapper over viper.BindPFlag for local FlagSet
//
// if flags is nil, the cmd.Flags() will be used
func ViperBindPFlagSet(cmd *cobra.Command, flags *pflag.FlagSet) {
	if flags == nil {
		flags = cmd.Flags()
	}
	flags.VisitAll(func(flag *pflag.Flag) {
		_ = viper.BindPFlag(PrefixKey(cmd, flag.Name), flag)
	})
}

// ViperBindPersistentPFlag is a convenience wrapper over viper.BindPFlag for persistent flags
func ViperBindPersistentPFlag(cmd *cobra.Command, name string) {
	_ = viper.BindPFlag(PrefixKey(cmd, name), cmd.PersistentFlags().Lookup(name))
}

// ViperGetString is a convenience wrapper that returns the value associated with the key as a string.
func ViperGetString(cmd *cobra.Command, key string) string {
	return viper.GetViper().GetString(PrefixKey(cmd, key))
}

// ViperGetStringSlice is a convenience wrapper that returns the value associated with the key as a slice of strings.
func ViperGetStringSlice(cmd *cobra.Command, key string) []string {
	return viper.GetViper().GetStringSlice(PrefixKey(cmd, key))
}

// ViperGetStringMap is a convenience wrapper that returns the value associated with the key as a map of interfaces.
func ViperGetStringMap(cmd *cobra.Command, key string) map[string]interface{} {
	return viper.GetViper().GetStringMap(PrefixKey(cmd, key))
}

// ViperGetStringMapString is a convenience wrapper that returns the value associated with the key as a map of strings.
func ViperGetStringMapString(cmd *cobra.Command, key string) map[string]string {
	return viper.GetViper().GetStringMapString(PrefixKey(cmd, key))
}

// ViperGetStringMapStringSlice is a convenience wrapper that returns the value associated with the key as a map to a slice of strings.
func ViperGetStringMapStringSlice(cmd *cobra.Command, key string) map[string][]string {
	return viper.GetViper().GetStringMapStringSlice(PrefixKey(cmd, key))
}

// ViperGetInt is a convenience wrapper that returns the value associated with the key as an integer.
func ViperGetInt(cmd *cobra.Command, key string) int {
	return viper.GetViper().GetInt(PrefixKey(cmd, key))
}

// ViperGetFloat64 is a convenience wrapper that returns the value associated with the key as a float64.
func ViperGetFloat64(cmd *cobra.Command, key string) float64 {
	return viper.GetViper().GetFloat64(PrefixKey(cmd, key))
}

// ViperGetTime is a convenience wrapper that returns the value associated with the key as time.
func ViperGetTime(cmd *cobra.Command, key string) time.Time {
	return viper.GetViper().GetTime(PrefixKey(cmd, key))
}

// ViperGetDuration is a convenience wrapper that returns the value associated with the key as a duration.
func ViperGetDuration(cmd *cobra.Command, key string) time.Duration {
	return viper.GetViper().GetDuration(PrefixKey(cmd, key))
}

// ViperGetBool is a convenience wrapper that returns the value associated with the key as a boolean.
func ViperGetBool(cmd *cobra.Command, key string) bool {
	return viper.GetViper().GetBool(PrefixKey(cmd, key))
}

// ViperGetSizeInBytes is a convenience wrapper that returns the size of the value associated with the given key
func ViperGetSizeInBytes(cmd *cobra.Command, key string) uint {
	return viper.GetViper().GetSizeInBytes(PrefixKey(cmd, key))
}

// ViperIsSet is a convenience wrapper returning true if a key is set. Case insensitive for keys.
func ViperIsSet(cmd *cobra.Command, key string) bool {
	return viper.GetViper().IsSet(PrefixKey(cmd, key))
}

// ViperSet is a convenience wrapper setting an override value for specified key
func ViperSet(cmd *cobra.Command, key, value string) {
	viper.GetViper().Set(PrefixKey(cmd, key), value)
}
