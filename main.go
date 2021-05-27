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

	"github.com/hjson/hjson-go"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/alphaflow/injector/gcp"
	"github.com/alphaflow/injector/pkg/jsonutil"
	"github.com/alphaflow/injector/pkg/stringsutil"
)

const (
	appName             = "inject"
	ashOutputFormatter  = `%s="%s"`
	bashOutputFormatter = `export %s="%s"`
	jsonIndent          = `    `
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
)

func main() {
	app := &cli.App{
		Name:                   appName,
		Usage:                  "Handle signals and inject environment variables from GCP secret manager.",
		Action:                 run,
		Version:                Version,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "key-file",
				Aliases:  []string{"k"},
				Usage:    "Path to file containing JSON format service account key.",
				Required: true,
			},
			&cli.BoolFlag{
				Name:     "format-ash",
				Aliases:  []string{"a"},
				Usage:    "Parse secret contents and convert to ash (shell) environment settings.",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "format-bash",
				Aliases:  []string{"b"},
				Usage:    "Parse secret contents and convert to bash (shell) environment settings.",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "format-json",
				Aliases:  []string{"j"},
				Usage:    "Parse secret contents and convert from hJSON to JSON.",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "format-raw",
				Aliases:  []string{"r"},
				Usage:    "Output unparsed secret contents. This will likely be hJSON or JSON.",
				Required: false,
			},
			&cli.BoolFlag{
				Name:    "preserve-env",
				Aliases: []string{"E"},
				Usage:   "Pass environment variables from parent OS into command shell. (default: false)",
			},
			&cli.StringFlag{
				Name:     "output-file",
				Aliases:  []string{"o"},
				Usage:    "Write output to file. Default is stdout; passing - also represents stdout.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "project",
				Aliases:  []string{"p"},
				Usage:    "GCP project id.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "secret-name",
				Usage:    "Name of secret containing environment variables and values.",
				Aliases:  []string{"S"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "secret-version",
				Usage:    "Version of secret containing environment variables and values. (default: latest)",
				Aliases:  []string{"V"},
				Required: false,
			},
		},
	}

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version: %s\n", Version)
		fmt.Printf("  build date: %s\n", BuildDate)
		fmt.Printf("  commit: %s\n", GitCommit)
		fmt.Printf("  branch: %s\n", GitBranch)
		fmt.Printf("  platform: %s\n", Platform)
		fmt.Printf("  built with: %s\n", runtime.Version())
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// run is the app main loop. Further branching will incur in this function to direct operations based on cli options.
func run(ctx *cli.Context) error {
	var buf bytes.Buffer

	// Disallow conflicting format options.
	conflict := 0
	if ctx.Bool("json") {
		conflict++
	}
	if ctx.Bool("raw") {
		conflict++
	}
	if conflict > 1 {
		return cli.Exit("multiple output formats are not supported", 1)
	}

	// Fetch the secret manager document content and copy to a buffer.
	if err := gcp.FetchSecretDocument(ctx, &buf); err != nil {
		log.Fatal(err)
	}

	// Set the output file to either stdout (default) or an actual file.
	outputFile := os.Stdout
	if !stringsutil.IsBlank(ctx.String("output-file")) && ctx.String("output-file") != "-" {
		var err error
		outputFile, err = os.Create(ctx.String("output-file"))
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			_ = outputFile.Close()
		}()
	}

	if ctx.Bool("format-json") {
		return outputJSON(ctx, &buf, outputFile)
	} else if ctx.Bool("format-raw") {
		return outputRaw(ctx, &buf, outputFile)
	} else if ctx.Bool("format-ash") {
		return outputAshEnv(ctx, &buf, outputFile)
	} else if ctx.Bool("format-bash") {
		return outputBashEnv(ctx, &buf, outputFile)
	}

	if err := runCommand(ctx, &buf, ctx.Args().Slice()); err != nil {
		log.Fatal(err)
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
	if len(commandWithArgs[0]) > 1 {
		args = commandWithArgs[1:]
	}

	// Define an exec command (with arguments), setup environment variables (passing through current environment
	// variables only if enabled), and rebind its stdout and stdin to the respective os streams.
	cmd := exec.Command(command, args...)
	cmd.Env = []string{}
	if ctx.Bool("preserve-env") {
		cmd.Env = os.Environ()
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Create a dedicated pidgroup used to forward signals to the main process and its children.
	// TODO: Add signal support.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var err error
	var data *map[string]interface{}
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

	err = cmd.Wait()
	if err != nil {
		log.WithError(err).Error("failed to wait for command to complete")
		return err
	}

	return nil
}

// convertMapToKeyValueList converts the parsed secret manager document environment variables to an array of key/value
// strings. This format is suitable for input to the `cmd.Env` string array value.
func convertMapToKeyValueList(ctx *cli.Context, data *map[string]interface{}) ([]string, error) {
	if ctx == nil {
		log.Fatal(errors.New("invalid context"))
	}

	if data == nil {
		log.Fatal(errors.New("invalid environment map"))
	}

	var jsonBytes []byte
	var err error
	if jsonBytes, err = json.Marshal(data); err != nil {
		log.Fatal(err)
	}

	return jsonutil.Flatten(jsonBytes, "environment", ashOutputFormatter), nil
}

// outputAshEnv writes the secret manager document contents as ash shell environment variables to the specified
// io.Writer.
func outputAshEnv(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer) error {
	return outputShell(ctx, buffer, writer, ashOutputFormatter)
}

// outputBashEnv writes the secret manager document contents as bash shell environment variables to the specified
// io.Writer.
func outputBashEnv(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer) error {
	return outputShell(ctx, buffer, writer, bashOutputFormatter)
}

// outputShell writes the secret manager document contents as shell environment variables, formatted with the given
// line formatter string, to the specified io.Writer.
func outputShell(ctx *cli.Context, buffer *bytes.Buffer, writer io.Writer, formatter string) error {
	if ctx == nil {
		log.Fatal(errors.New("invalid context"))
	}

	if buffer == nil {
		log.Fatal(errors.New("invalid buffer"))
	}

	var err error
	var data *map[string]interface{}
	if data, err = parseHJSON(ctx, buffer); err != nil {
		return err
	}

	var jsonBytes []byte
	if jsonBytes, err = json.Marshal(data); err != nil {
		log.Fatal(err)
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
	var data *map[string]interface{}
	if data, err = parseHJSON(ctx, buffer); err != nil {
		return err
	}

	var prettyJSON []byte
	if prettyJSON, err = json.MarshalIndent(data, "", jsonIndent); err != nil {
		log.Fatal(err)
	}

	prettyJSON = jsonutil.ConvertUnicodeToAscii(prettyJSON)

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
		log.Fatal(errors.New("invalid context"))
	}

	if buffer == nil {
		log.Fatal(errors.New("invalid buffer"))
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
func parseHJSON(ctx *cli.Context, buffer *bytes.Buffer) (*map[string]interface{}, error) {
	if ctx == nil {
		log.Fatal(errors.New("invalid context"))
	}

	if buffer == nil {
		log.Fatal(errors.New("invalid buffer"))
	}

	// marshal JSON to a generic map
	data := make(map[string]interface{}, 0)
	if err := hjson.Unmarshal(buffer.Bytes(), &data); err != nil {
		log.Fatal(err)
	}

	return &data, nil
}
