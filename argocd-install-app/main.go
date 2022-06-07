package main

import (
	"fmt"
	ga "github.com/sethvargo/go-githubactions"
	"os"
	"strings"
)

type Config struct {
	GrpcWeb   string
	Server    string
	AuthToken string
	Ticket    string
	Version   string
	Revision  string
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

	action.AddStepSummary(fmt.Sprintf("ArgoCD Ticket: %4s", cfg.Ticket))
	action.AddStepSummary(fmt.Sprintf("ArgoCD Version: %4s", cfg.Version))
	action.AddStepSummary(fmt.Sprintf("ArgoCD Revision: %4s", cfg.Revision))

	return nil
}

func newCfgFromInputs(action *ga.Action) (*Config, error) {
	for _, s := range os.Environ() {
		action.Infof("env: %s", s)
		parts := strings.Split(s, "=")
		if len(parts) == 2 {
			action.SetEnv(parts[0], parts[1])
		}
	}

	c := &Config{
		GrpcWeb:   action.GetInput("grpc-web"),
		Server:    action.GetInput("server"),
		AuthToken: action.GetInput("auth-token"),
		Ticket:    action.GetInput("ticket"),
		Version:   action.GetInput("version"),
		Revision:  action.GetInput("revision"),
	}

	return c, nil
}

func main() {
	action := ga.New()
	err := run(action)
	if err != nil {
		action.Fatalf("%v", err)
	}
}
