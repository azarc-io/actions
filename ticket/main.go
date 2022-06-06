package main

import (
	"flag"
	"fmt"
	ga "github.com/sethvargo/go-githubactions"
	"regexp"
)

type Config struct {
	Ref         string
	Prefix      string
	Environment string
	Revision    string
}

func newCfgFromInputs(action *ga.Action) (*Config, error) {
	c := Config{
		Ref:         action.GetInput("ref"),
		Prefix:      action.GetInput("prefix"),
		Environment: action.GetInput("environment"),
	}

	if c.Prefix == "" {
		action.Fatalf("prefix is required")
	}

	if c.Ref == "" {
		action.Fatalf("ref is required")
	}

	if ctx, err := action.Context(); err != nil {
		return nil, err
	} else {
		c.Revision = ctx.SHA
	}

	return &c, nil
}

func extractIssue(args *Config) string {
	return regexp.MustCompile(args.Prefix + "\\-\\d+").FindString(
		fmt.Sprintln(
			args.Ref,
		),
	)
}

func run(action *ga.Action) error {
	var (
		cfg *Config
		err error
	)

	cfg, err = newCfgFromInputs(action)
	if err != nil {
		ga.Fatalf("%v", err)
	}

	issue := extractIssue(cfg)

	if issue == "" {
		action.Fatalf("could not extract issue from ref: %s", cfg.Ref)
	}

	action.SetOutput("revision", cfg.Revision)
	action.SetOutput("ticket", extractIssue(cfg))
	action.AddStepSummary(fmt.Sprintf("Ticket: %4s", extractIssue(cfg)))
	action.AddStepSummary(fmt.Sprintf("Ref: %4s", cfg.Ref))
	action.AddStepSummary(fmt.Sprintf("Prefix: %4s", cfg.Prefix))
	action.AddStepSummary(fmt.Sprintf("Revision: %4s", cfg.Revision))
	action.AddStepSummary(fmt.Sprintf("Environment: %4s", cfg.Environment))

	return nil
}

func main() {
	flag.Parse()
	action := ga.New()
	err := run(action)
	if err != nil {
		action.Fatalf("%v", err)
	}
}
