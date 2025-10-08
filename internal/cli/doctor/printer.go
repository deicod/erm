package doctor

import (
	"fmt"
	"io"
	"strings"
)

// Printer formats doctor output.
type Printer struct {
	out io.Writer
}

// NewPrinter constructs a printer that writes to out.
func NewPrinter(out io.Writer) *Printer {
	return &Printer{out: out}
}

// PrintHeader renders the command heading.
func (p *Printer) PrintHeader(title string) {
	fmt.Fprintf(p.out, "%s\n\n", title)
}

// PrintSystem reports system metadata.
func (p *Printer) PrintSystem(os, arch, goVersion string) {
	fmt.Fprintf(p.out, "System: %s/%s\n", os, arch)
	if goVersion != "" {
		fmt.Fprintf(p.out, "Go: %s\n", goVersion)
	}
	fmt.Fprintln(p.out)
}

// PrintCheck prints the outcome of a single check.
func (p *Printer) PrintCheck(res Result) {
	fmt.Fprintf(p.out, "[%s] %s", strings.ToUpper(string(res.Status)), res.Name)
	if res.Details != "" {
		fmt.Fprintf(p.out, " - %s", res.Details)
	}
	fmt.Fprintln(p.out)
}

// Summary prints aggregate status counts.
func (p *Printer) Summary(results []Result) {
	var okCount, warnCount, errCount int
	for _, res := range results {
		switch res.Status {
		case StatusOK:
			okCount++
		case StatusWarn:
			warnCount++
		case StatusError:
			errCount++
		}
	}
	fmt.Fprintln(p.out)
	fmt.Fprintf(p.out, "Summary: %d ok, %d warnings, %d errors\n", okCount, warnCount, errCount)
	if errCount > 0 {
		fmt.Fprintln(p.out, "Resolve errors above then re-run 'erm doctor'.")
	}
}
