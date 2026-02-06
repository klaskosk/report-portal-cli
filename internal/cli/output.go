package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

// Printer formats command output according to the chosen output mode.
type Printer struct {
	format string
	out    io.Writer
	errOut io.Writer
}

func newPrinter(format string) *Printer {
	return &Printer{
		format: format,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
}

// PrintTable writes rows as a tab-aligned table. headers is the first row; each element of rows
// is a slice of column values for one row.
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	tabWriter := tabwriter.NewWriter(p.out, 0, 4, 2, ' ', 0)

	_, err := fmt.Fprintln(tabWriter, strings.Join(headers, "\t"))
	if err != nil {
		return fmt.Errorf("writing table header: %w", err)
	}

	for _, row := range rows {
		_, err = fmt.Fprintln(tabWriter, strings.Join(row, "\t"))
		if err != nil {
			return fmt.Errorf("writing table row: %w", err)
		}
	}

	err = tabWriter.Flush()
	if err != nil {
		return fmt.Errorf("flushing table: %w", err)
	}

	return nil
}

// PrintJSON writes value as indented JSON.
func (p *Printer) PrintJSON(value any) error {
	enc := json.NewEncoder(p.out)
	enc.SetIndent("", "  ")

	err := enc.Encode(value)
	if err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}

	return nil
}

// PrintTemplate parses tmpl as a Go text/template and executes it against value.
func (p *Printer) PrintTemplate(tmpl string, value any) error {
	templateInstance, err := template.New("output").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	err = templateInstance.Execute(p.out, value)
	if err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	_, err = fmt.Fprintln(p.out)
	if err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// PrintError formats err according to the current output mode. JSON mode writes a JSON object
// to stdout; other modes write plain text to stderr.
func (p *Printer) PrintError(err error) {
	if p.isJSON() {
		_ = p.PrintJSON(map[string]string{"error": err.Error()})

		return
	}

	_, _ = fmt.Fprintf(p.errOut, "Error: %s\n", err)
}

// Print dispatches to the appropriate output method based on the configured format.
// tableHeaders and tableRows are used for table mode; value is used for JSON and go-template modes.
func (p *Printer) Print(value any, tableHeaders []string, tableRows [][]string) error {
	if p.isJSON() {
		return p.PrintJSON(value)
	}

	if tmpl, ok := p.goTemplateString(); ok {
		return p.PrintTemplate(tmpl, value)
	}

	if p.isTable() {
		return p.PrintTable(tableHeaders, tableRows)
	}

	return fmt.Errorf("unknown output format: %q", p.format)
}

func (p *Printer) isJSON() bool {
	return p.format == "json"
}

func (p *Printer) isTable() bool {
	return p.format == "" || p.format == "table"
}

func (p *Printer) goTemplateString() (string, bool) {
	if after, ok := strings.CutPrefix(p.format, "go-template="); ok {
		return after, true
	}

	return "", false
}
