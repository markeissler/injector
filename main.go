package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	cliTemplate "text/template"

	"github.com/hjson/hjson-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/markeissler/injector/gcp"
	"github.com/markeissler/injector/pkg/jsonutil"
	"github.com/markeissler/injector/pkg/numericutil"
	"github.com/markeissler/injector/pkg/signal"
	"github.com/markeissler/injector/pkg/stringutil"
	"github.com/markeissler/injector/template"
)

const (
	appName                     = "inject"
	exportedOutputFormatter     = `export %s="%s"`
	unexportedOutputFormatter   = `%s="%s"`
	unquotedOutputFormatter     = `%s=%s`
	jsonIndent                  = `    `
	envVarInjectorKeyValue      = "INJECTOR_KEY_VALUE"
	envVarInjectorProject       = "INJECTOR_PROJECT"
	envVarInjectorSecretName    = "INJECTOR_SECRET_NAME"
	envVarInjectorSecretVersion = "INJECTOR_SECRET_VERSION"
)

var (
	// Version contains the current Version.
	Version = "dev"
	// BuildDate contains a string with the build BuildDate.
	BuildDate = "unknown"
	// GitCommit git commit sha
	GitCommit = "dirty"
	// GitBranch git branch
	GitBranch = "dirty"
	// Platform OS/ARCH
	Platform = ""
	// Logger
	log = logrus.New()
)

func main() {
	app := &cli.App{
		Name:                   appName,
		HelpName:               appName,
		Usage:                  "Handle signals and inject environment variables from GCP secret manager.",
		Action:                 run,
		Version:                Version,
		UseShortOptionHandling: true,
		Flags:                  flags(),
	}

	cli.AppHelpTemplate = template.AppHelpTemplate()
	cli.HelpPrinter = func(out io.Writer, templ string, data interface{}) {
		funcMap := cliTemplate.FuncMap{
			"stripDefault": template.StripDefault,
		}
		cli.HelpPrinterCustom(out, templ, data, funcMap)
	}

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(os.Stdout, "version: %s\n", Version)
		fmt.Fprintf(os.Stdout, "  build date: %s\n", BuildDate)
		fmt.Fprintf(os.Stdout, "  commit: %s\n", GitCommit)
		fmt.Fprintf(os.Stdout, "  branch: %s\n", GitBranch)
		fmt.Fprintf(os.Stdout, "  platform: %s\n", Platform)
		fmt.Fprintf(os.Stdout, "  built with: %s\n", runtime.Version())
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// debug outputs version information, resolved inputs from cli options and environment variables to the specified
// io.Writer.
func debug(ctx *cli.Context, writer io.Writer) {
	cli.ShowVersion(ctx)

	for _, flag := range ctx.App.Flags {
		for _, name := range flag.Names() {
			if len(name) == 1 {
				// skip aliases
				continue
			}

			value := ctx.String(name)
			if stringutil.IsBlank(value) {
				value = "<NOT SET>"
			}
			fmt.Fprintf(writer, "%s: %s\n", name, value)
		}
	}

	for i, a := range ctx.Args().Slice() {
		fmt.Fprintf(writer, "  [%d]: %v\n", i, a)
	}
}

// flags defines all of the option flags and corresponding environment variables (if applicable) for the app.
// nolint:funlen
func flags() []cli.Flag {
	return []cli.Flag{
		// key-file represents the path to a text file containing a JSON-formatted service account key for accessing the
		// target secret manager document. It is an error to specify both `key-file` and `key-value`.
		&cli.StringFlag{
			Name:     "key-file",
			Aliases:  []string{"k"},
			Usage:    "Path to file containing JSON format service account key.",
			Required: false,
		},
		// key-value is a text string containing a base64 encoded JSON-formatted service account key for accessing the
		// target secret manager document. It is an error to specify both `key-file` and `key-value`. This value can be
		// set via the cli or via an environment variable.
		&cli.StringFlag{
			Name:     "key-value",
			Aliases:  []string{"K"},
			Usage:    "Base64 encoded string containing JSON format service account key.",
			Required: false,
			EnvVars:  []string{envVarInjectorKeyValue},
		},
		// format-shell outputs contents from the secret document as a list of exported shell key/value settings. A
		// typical use case would be to write the output to a file and then `source` it elsewhere.
		&cli.BoolFlag{
			Name:     "format-shell",
			Aliases:  []string{"e"},
			Usage:    "Parse secret contents and convert to exported shell key/value settings.",
			Required: false,
		},
		// format-shell-unexported outputs contents from the secret document as a list of shell key/value settings. A
		// typical use case would be to write the output to a file and then `source` it elsewhere.
		&cli.BoolFlag{
			Name:     "format-shell-unexported",
			Aliases:  []string{"u"},
			Usage:    "Parse secret contents and convert to unexported shell key/value settings.",
			Required: false,
		},
		// format-json outputs contents from the secret document as a standard JSON object.
		&cli.BoolFlag{
			Name:     "format-json",
			Aliases:  []string{"j"},
			Usage:    "Parse secret contents and convert from hJSON to JSON.",
			Required: false,
		},
		// format-raw outputs contents from the secret document as returned by the secret manager. The format returned
		// should be either hJSON (human JSON) or standard JSON.
		&cli.BoolFlag{
			Name:     "format-raw",
			Aliases:  []string{"r"},
			Usage:    "Output unparsed secret contents. This will likely be hJSON or JSON.",
			Required: false,
		},
		// ignore would generally be used for deployments where the command line includes one or more secret retrieval
		// options (for instance, in a container run command) and other values are intended to be pulled from env vars
		// but could be missing while debugging locally. Specifying this option would
		&cli.BoolFlag{
			Name:     "ignore",
			Aliases:  []string{"i"},
			Usage:    "Ignore missing secret options.",
			Required: false,
		},
		// ignore-preserve-env is different from supplying -i and -E in that specifying this option will only pass
		// parent environment variables if secret retrieval options are missing (i.e. an incomplete set of options were
		// specified). In contrast, specifying -E will always pass parent environment variables.
		&cli.BoolFlag{
			Name:     "ignore-preserve-env",
			Aliases:  []string{"I"},
			Usage:    "Ignore missing secret options, pass environment variables from parent OS into command shell.",
			Required: false,
		},
		// preserve-env will pass through environment varisables from the parent to the child process (that is, the
		// process that is specified as the command to run).
		&cli.BoolFlag{
			Name:     "preserve-env",
			Aliases:  []string{"E"},
			Usage:    "Pass environment variables from parent OS into command shell.",
			Required: false,
		},
		// output-file sets the output destination which is stdout by default but can also be set to a file path. The
		// "-" character, as the path, also identifies stdout as the destination.
		&cli.StringFlag{
			Name:     "output-file",
			Aliases:  []string{"o"},
			Usage:    `Write output to file. Default is stdout; passing "-" also represents stdout.`,
			Required: false,
		},
		// project sets the GCP project id in which the secret manager document is stored. This value can be set via the
		// cli or via an environment variable.
		&cli.StringFlag{
			Name:     "project",
			Aliases:  []string{"p"},
			Usage:    "GCP project id.",
			Required: false,
			EnvVars:  []string{envVarInjectorProject},
		},
		// secret-name sets the GCP secret manager document name which identifies the specific document to retrieve.
		// This value can be set via the cli or via an environment variable.
		&cli.StringFlag{
			Name:     "secret-name",
			Usage:    "Name of secret containing environment variables and values.",
			Aliases:  []string{"S"},
			Required: false,
			EnvVars:  []string{envVarInjectorSecretName},
		},
		// secret-version set the version (revision) of the GCP secret manager document to retrieve. This setting is
		// strictly option and the behavior is to retrieve the `latest` version of the named secret. Beware that setting
		// a non-existent version will return an empty value (this is desired behavior).
		&cli.StringFlag{
			Name:     "secret-version",
			Usage:    `Version of secret containing environment variables and values. ("latest" if not specified)`,
			Aliases:  []string{"V"},
			Required: false,
			EnvVars:  []string{envVarInjectorSecretVersion},
		},
		// debug enables the output of debugging information which is specifically helpful in identifying misconfigured
		// and possibly conflicting settings.
		&cli.BoolFlag{
			Name:     "debug",
			Usage:    "Show debug information.",
			Aliases:  []string{"d"},
			Required: false,
		},
	}
}

// hasConflictingOptions checks for options that may conflict. A conflict exists when multiple options enable similar
// functionality and/or when an option is configured via an environment variable while a conflicting option is set from
// a cli option flag (if specifying both options via cli option flag, for instance, a conflict would also occur).
func hasConflictingOptions(ctx *cli.Context) (bool, error) {
	// Disallow conflicting format options.
	if numericutil.BoolToInt(ctx.Bool("format-shell"))+numericutil.BoolToInt(ctx.Bool("format-shell-unexported"))+
		numericutil.BoolToInt(ctx.Bool("format-json"))+numericutil.BoolToInt(ctx.Bool("format-raw")) > 1 {
		return true, errors.New("multiple output formats are not supported")
	}

	// Disallow conflicting environment pass through options.
	if numericutil.BoolToInt(ctx.Bool("preserve-env"))+numericutil.BoolToInt(ctx.Bool("ignore-preserve-env")) > 1 {
		return true, errors.New("multiple preserve environment options are not supported")
	}

	// Disallow conflicting key source options.
	if numericutil.StringToBoolInt(ctx.String("key-file"))+numericutil.StringToBoolInt(ctx.String("key-value")) > 1 {
		return true, errors.New("multiple key source formats are not supported")
	}

	return false, nil
}

// hasMissingRetrievalOptions checks for an incomplete set of secret retrieval options. If at least one of the options
// has been specified then all dependent options need to be specified as well.
//
// Dependencies:
//	- (key-file or key-value) + project + secret-name
//	- secret-version + (key-file or key-value) + project + secret-name
//
// The `secret-version` option cannot be specified without also specifying all other dependent options.
func hasMissingRetrievalOptions(ctx *cli.Context) (bool, error) {
	minimumCount := 3
	if !stringutil.IsBlank(ctx.String("secret-version")) {
		minimumCount++
	}

	// Disallow only some of the secret retrieval options to be defined.
	actualCount := numericutil.BoolToInt(
		numericutil.StringToBool(ctx.String("key-file")) || numericutil.StringToBool(ctx.String("key-value"))) +
		numericutil.StringToBoolInt(ctx.String("project")) + numericutil.StringToBoolInt(ctx.String("secret-name")) +
		numericutil.StringToBoolInt(ctx.String("secret-version"))

	if actualCount > 0 && actualCount < minimumCount {
		return true, errors.New("missing dependencies for secret retrieval options")
	}

	return false, nil
}

// run is the app main loop. Further branching will incur in this function to direct operations based on cli options.
func run(ctx *cli.Context) error {
	var buf bytes.Buffer

	// Output debug information and continue.
	if ctx.Bool("debug") {
		debug(ctx, os.Stdout)
	}

	// Make sure potentially conflicting options are not set.
	if bad, err := hasConflictingOptions(ctx); bad {
		return err
	}

	// Make sure all required options are set if fetching a secret manager document.
	if bad, err := hasMissingRetrievalOptions(ctx); bad {
		return err
	}

	// Fetch the secret manager document content and copy to a buffer.
	if wantsToPullSecret(ctx) {
		if err := gcp.FetchSecretDocument(ctx, &buf); err != nil && !wantsToIgnorePullSecretFailures(ctx) {
			return err
		}
	}

	// Set the output file to either stdout (default) or an actual file.
	outputFile := os.Stdout
	if !stringutil.IsBlank(ctx.String("output-file")) && ctx.String("output-file") != "-" {
		var err error
		outputFile, err = os.Create(ctx.String("output-file"))
		if err != nil {
			return err
		}
		defer func() {
			_ = outputFile.Close()
		}()
	}

	if ctx.Bool("format-json") {
		return outputJSON(ctx, &buf, outputFile)
	} else if ctx.Bool("format-raw") {
		return outputRaw(ctx, &buf, outputFile)
	} else if ctx.Bool("format-shell") {
		return outputShellExported(ctx, &buf, outputFile)
	} else if ctx.Bool("format-shell-unexported") {
		return outputShellUnexported(ctx, &buf, outputFile)
	}

	if err := runCommand(ctx, &buf, ctx.Args().Slice()); err != nil {
		return err
	}

	return nil
}

// runCommand runs the intended command in the default user shell with injected environment variables.
func runCommand(ctx *cli.Context, buf *bytes.Buffer, commandWithArgs []string) error {
	var command string
	var args []string

	if len(commandWithArgs) == 0 {
		log.Warn("no command specified")
		return nil
	}

	command = commandWithArgs[0]
	if !stringutil.IsBlank(command) && len(commandWithArgs) > 1 {
		args = commandWithArgs[1:]
	}

	// Define an exec command (with arguments), setup environment variables (passing through current environment
	// variables only if enabled), and rebind its stdout and stdin to the respective os streams.
	cmd := exec.Command(command, args...)
	cmd.Env = []string{}
	if ctx.Bool("preserve-env") || (ctx.Bool("ignore-preserve-env") && buf.Len() == 0) {
		cmd.Env = os.Environ()
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Create a dedicated pidgroup used to forward signals to the main process and its children.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var err error
	var data map[string]interface{}
	if data, err = parseHJSON(ctx, buf); err != nil {
		return err
	}

	var envList []string
	envList, err = convertMapToKeyValueList(ctx, data)
	if err != nil {
		log.WithError(err).Error("failed to resolve secrets")
	}
	cmd.Env = append(cmd.Env, envList...)

	err = cmd.Start()
	if err != nil {
		log.WithError(err).Error("failed to start command")
		return err
	}

	// Trap signals and forward to the child process.
	signal.ForwardToPid(cmd.Process.Pid, log, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	err = cmd.Wait()
	if err != nil {
		log.WithError(err).Error("failed to wait for command to complete")
		return err
	}

	return nil
}

// convertMapToKeyValueList converts the parsed secret manager document environment variables to an array of key/value
// strings. This format is suitable for input to the `cmd.Env` string array value.
func convertMapToKeyValueList(ctx *cli.Context, data map[string]interface{}) ([]string, error) {
	if ctx == nil {
		return []string{}, errors.New("invalid context")
	}

	if data == nil {
		return []string{}, errors.New("invalid environment map")
	}

	var jsonBytes []byte
	var err error
	if jsonBytes, err = json.Marshal(data); err != nil {
		return []string{}, err
	}

	return jsonutil.Flatten(jsonBytes, "environment", unquotedOutputFormatter), nil
}

// outputShellExported writes the secret manager document contents as exported shell key/value variables to the
// specified io.Writer.
func outputShellExported(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer) error {
	return outputShell(ctx, buffer, writer, exportedOutputFormatter)
}

// outputShellUnexported writes the secret manager document contents as unexported shell key/value variables to the
// specified io.Writer.
func outputShellUnexported(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer) error {
	return outputShell(ctx, buffer, writer, unexportedOutputFormatter)
}

// outputShell writes the secret manager document contents as shell environment variables, formatted with the given
// line formatter string, to the specified io.Writer.
func outputShell(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer, formatter string) error {
	if ctx == nil {
		return errors.New("invalid context")
	}

	if buffer == nil {
		return errors.New("invalid buffer")
	}

	var err error
	var data map[string]interface{}
	if data, err = parseHJSON(ctx, buffer); err != nil {
		return err
	}

	var jsonBytes []byte
	if jsonBytes, err = json.Marshal(data); err != nil {
		return err
	}

	list := jsonutil.Flatten(jsonBytes, "environment", formatter)

	for _, v := range list {
		fmt.Fprintf(writer, "%s\n", v)
	}

	return nil
}

// outputJSON write the secret manager document contents as JSON to the specified io.Writer.
func outputJSON(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer) error {
	if ctx == nil {
		log.Fatal(errors.New("invalid context"))
	}

	if buffer == nil {
		log.Fatal(errors.New("invalid buffer"))
	}

	var err error
	var data map[string]interface{}
	if data, err = parseHJSON(ctx, buffer); err != nil {
		return err
	}

	var prettyJSON []byte
	if prettyJSON, err = json.MarshalIndent(data, "", jsonIndent); err != nil {
		log.Fatal(err)
	}

	prettyJSON = jsonutil.ConvertUnicodeToASCII(prettyJSON)

	n, err := writer.Write(prettyJSON)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if n != len(prettyJSON) {
		fmt.Println("failed to write data")
		os.Exit(1)
	}

	return nil
}

// outputRaw write the raw secret manager document contents to the specified io.Writer.
func outputRaw(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer) error {
	if ctx == nil {
		return errors.New("invalid context")
	}

	if buffer == nil {
		return errors.New("invalid buffer")
	}

	n, err := writer.Write(buffer.Bytes())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if n != len(buffer.Bytes()) {
		fmt.Println("failed to write data")
		os.Exit(1)
	}

	return nil
}

// parseHJSON parses the raw secret manager document contents in JSON or HJSON content into a map.
func parseHJSON(ctx *cli.Context, buffer *bytes.Buffer) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	if ctx == nil {
		return data, errors.New("invalid context")
	}

	if buffer == nil {
		return data, errors.New("invalid buffer")
	}

	if err := hjson.Unmarshal(buffer.Bytes(), &data); err != nil {
		log.Fatal(err)
	}

	return data, nil
}

// wantsToPullSecret checks if supplied options indicate the user wants to retrieve a secret manager document.
func wantsToPullSecret(ctx *cli.Context) bool {
	// We only need to check if one of the options that would be needed to pull a secret is defined.
	return numericutil.StringToBool(ctx.String("project"))
}

// wantsToIgnorePullSecretFailures checks if supplied options indicate the user wants to ignore any errors encountered
// when attempting to retrieve a secret manager document.
func wantsToIgnorePullSecretFailures(ctx *cli.Context) bool {
	return ctx.Bool("ignore") || ctx.Bool("ignore-preserve-env")
}
