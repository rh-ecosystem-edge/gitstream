package cli

import (
	"flag"
	"time"
)

const dayLayout = "2006-01-02"

type Day struct {
	t *time.Time
}

func (d *Day) Set(s string) error {
	t, err := time.Parse(dayLayout, s)
	if err != nil {
		return err
	}

	d.t = &t

	return nil
}

func (d *Day) String() string {
	t := d.t

	if t == nil {
		t = &time.Time{}
	}

	return t.Format(dayLayout)
}

func (d *Day) Time() *time.Time {
	return d.t
}

type CommandLine struct {
	DownstreamRepoName string
	DownstreamRepoPath string
	DownstreamSince    Day
	DryRun             bool
	UpstreamRef        string
	UpstreamSince      Day
	UpstreamURL        string
}

func Parse(args []string) (*CommandLine, error) {
	cl := CommandLine{
		DownstreamSince: Day{},
		UpstreamSince:   Day{},
	}

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)

	fs.StringVar(&cl.DownstreamRepoPath, "downstream-repo-path", ".", "path to a local clone of the downstream repo")
	fs.Var(&cl.DownstreamSince, "downstream-since", "only look at downstream commits on or after that date")
	fs.StringVar(&cl.DownstreamRepoName, "downstream-repo-name", "", "the name of the downstream repo on GitHub, e.g. owner/repo")
	fs.BoolVar(&cl.DryRun, "dry-run", false, "do not create anything on GitHub")
	fs.Var(&cl.UpstreamSince, "upstream-since", "only look at upstream commits on or after that date")
	fs.StringVar(&cl.UpstreamRef, "upstream-ref", "main", "the name of the upstream reference")
	fs.StringVar(&cl.UpstreamURL, "upstream-url", "", "the path to the upstream URL")

	return &cl, fs.Parse(args[1:])
}
