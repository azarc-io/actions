package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/commander-cli/cmd"
	ga "github.com/sethvargo/go-githubactions"
	"html/template"
	"io"
	"k8s.io/apimachinery/pkg/util/yaml"
	"path"
	"strings"
	"time"
)

type Config struct {
	GrpcWeb   bool
	Server    string
	AuthToken string
	Ticket    string
	Version   string
	Revision  string
	Template  string
	Services  []string
	Workspace string
	client    apiclient.Client
	Action    string
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

	switch cfg.Action {
	case "create":
		return installAction(cfg, action)
	case "delete":
		return deleteAction(cfg, action)
	default:
		action.Fatalf("unknown action type: %s", cfg.Action)
	}
	return nil
}

func deleteAction(cfg *Config, action *ga.Action) error {
	var (
		issueLower = strings.ToLower(cfg.Ticket)
	)

	// app client
	closer, ac, err := cfg.client.NewApplicationClient()
	if err != nil {
		action.Fatalf("failed to create a project client: %s", err.Error())
	}
	defer func(closer io.Closer) {
		err := closer.Close()
		if err != nil {
			action.Fatalf("failed to close client: %s", err.Error())
		}
	}(closer)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	_, err = ac.Delete(ctx, &application.ApplicationDeleteRequest{
		Name:    stringRef(issueLower),
		Cascade: boolRef(true),
	})

	return err
}

func installAction(cfg *Config, action *ga.Action) error {
	var (
		issueLower = strings.ToLower(cfg.Ticket)
	)

	action.Infof("creating new project for [issue]: %s [grpc-web]: %v", issueLower, cfg.GrpcWeb)

	// load template
	file, err := readFile(path.Join(cfg.Workspace, cfg.Template))
	if err != nil {
		action.Fatalf("could not load template: %s", err.Error())
	}
	// parse template
	t, err := template.New("app").Parse(file)
	if err != nil {
		action.Fatalf("could not parse template: %s", err.Error())
	}
	// template model
	var tpl bytes.Buffer
	err = t.Execute(&tpl, map[string]string{
		"REVISION": cfg.Revision,
		"TICKET":   issueLower,
		"VERSION":  cfg.Version,
	})
	if err != nil {
		action.Fatalf("could not execute template: %s", err.Error())
	}
	// create application
	app := v1alpha1.Application{}
	err = yaml.Unmarshal(tpl.Bytes(), &app)
	if err != nil {
		return err
	}
	// app client
	closer, ac, err := cfg.client.NewApplicationClient()
	if err != nil {
		action.Fatalf("failed to create a project client: %s", err.Error())
	}
	defer func(closer io.Closer) {
		err := closer.Close()
		if err != nil {
			action.Fatalf("failed to close client: %s", err.Error())
		}
	}(closer)

	// create version for each service
	for _, s := range cfg.Services {
		entries := strings.Split(s, ":")
		if len(entries) != 2 {
			panic("service list should follow pattern of name:path")
		}
		name := entries[0]
		pth := entries[1]
		action.Infof("setting version for service %s => %s", pth, cfg.Version)
		app.Spec.Source.Helm.Parameters = append(app.Spec.Source.Helm.Parameters, v1alpha1.HelmParameter{
			Name:        fmt.Sprintf("versions.%s", name),
			Value:       cfg.Version,
			ForceString: true,
		})
	}

	b, _ := json.MarshalIndent(tpl, "", " ")
	action.Infof("template:\n %s", string(b))

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	_, err = ac.Create(ctx, &application.ApplicationCreateRequest{
		Application: app,
		Upsert:      boolRef(true),
		Validate:    boolRef(true),
	})
	if err != nil {
		action.Warningf("could not deploy app: %v", app)
		action.Fatalf("failed to deploy app: %s", err.Error())
	}

	_, err = ac.Sync(ctx, &application.ApplicationSyncRequest{
		Name:     stringRef(issueLower),
		Revision: cfg.Revision,
		DryRun:   false,
		Prune:    true,
		SyncOptions: &application.SyncOptions{
			Items: []string{
				"CreateNamespace=true",
			},
		},
	})
	if err != nil {
		action.Errorf("failed to sync app, it is probably already being synced: %s", err.Error())
	}

	time.Sleep(time.Second * 3)

	return execWait(fmt.Sprintf("argocd.argoproj.io/instance=%s", issueLower), cfg)
}

func execWait(label string, args *Config) error {
	c := cmd.NewCommand(
		fmt.Sprintf("argocd app wait -l %s --timeout 240 --health --sync --health --operation --auth-token %s --server %s --grpc-web", label, args.AuthToken, args.Server),
		cmd.WithStandardStreams, cmd.WithTimeout(time.Minute*4))
	return c.Execute()
}

func newCfgFromInputs(action *ga.Action) (*Config, error) {
	c := &Config{
		GrpcWeb:   action.GetInput("grpc_web") == "true",
		Server:    action.GetInput("server"),
		AuthToken: action.GetInput("auth_token"),
		Ticket:    action.GetInput("ticket"),
		Version:   action.GetInput("version"),
		Revision:  action.GetInput("revision"),
		Template:  action.GetInput("template"),
		Action:    action.GetInput("action"),
	}

	sl := action.GetInput("service_list")
	c.Services = strings.Split(strings.TrimSpace(sl), "\n")

	ctx, err := action.Context()
	if err != nil {
		return nil, err
	}
	c.Workspace = ctx.Workspace

	// create client
	client, err := apiclient.NewClient(&apiclient.ClientOptions{
		ServerAddr:   c.Server,
		AuthToken:    c.AuthToken,
		GRPCWeb:      c.GrpcWeb,
		HttpRetryMax: 3,
	})
	if err != nil {
		return nil, err
	}
	c.client = client

	return c, nil
}

func main() {
	action := ga.New()
	err := run(action)
	if err != nil {
		action.Fatalf("%v", err)
	}
}
