package template

import (
	"fmt"
	"regexp"

	"github.com/urfave/cli/v2"
)

// AppHelpTemplate returns the text template for the Default help topic.
func AppHelpTemplate() string {
	return `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText | nindent 3 | trim}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description | nindent 3 | trim}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{else}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{stripDefault $option}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`
}

// StripDefault is a custom template function that will strip occurrences of the `(default: <TEXT>)` pattern from the
// help output to avoid leaking sensitive values.
//
// The current distributed implementation of the help functionality inserts default values in the help description for
// each option (as applicable). Unfortunately, the downside to this is that defaults are read from hard-coded values
// but overridden by values that may be set in the environment (for options that support reading from env vars). While
// it's less likely that sensitive values would be hardcoded and compiled into the binary, it's more likely that these
// values would be set via env vars which means that at run time their values would be exposed to users that have the
// necessary level of access to run `inject -h`. A number of issues have been opened against the repo to provide a way
// of changing this behavior but no changes have been made to address the problem as of yet so we are just going to get
// around it with a custom `AppHelpTemplate` and passing this function to support the template.
//
// You will need to insert statements like the following before you call `app.Run()`:
//
// ```go
//	cli.AppHelpTemplate = template.AppHelpTemplate()
//	cli.HelpPrinter = func(out io.Writer, templ string, data interface{}) {
//		funcMap := tt.FuncMap{
//			"stripDefault": template.StripDefault,
//		}
//		cli.HelpPrinterCustom(out, templ, data, funcMap)
//	}
// ```
func StripDefault(v interface{}) string {
	r := regexp.MustCompile(`\(default:\s+.*\)\s*`)

	if _, ok := v.(cli.Flag); ok {
		return r.ReplaceAllString(v.(cli.Flag).String(), "")
	}

	return fmt.Sprintf("%s", v.(cli.Flag).String())
}
