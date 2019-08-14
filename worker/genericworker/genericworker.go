package genericworker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/taskcluster/taskcluster-worker-runner/protocol"
	"github.com/taskcluster/taskcluster-worker-runner/runner"
	"github.com/taskcluster/taskcluster-worker-runner/worker/worker"
)

type genericworkerConfig struct {
	Path       string
	ConfigPath string
}

type genericworker struct {
	runnercfg *runner.RunnerConfig
	wicfg     genericworkerConfig
	cmd       *exec.Cmd
}

func (d *genericworker) ConfigureRun(run *runner.Run) error {
	var err error

	// copy some values from the provisioner metadata, if they are set; if not,
	// generic-worker will fall back to defaults
	for cfg, md := range map[string]string{
		// generic-worker config : providerMetadata
		"host":           "public-hostname",
		"publicIp":       "public-ipv4",
		"privateIp":      "local-ipv4",
		"workerNodeType": "instance-type",
		"instanceType":   "instance-type",
		"instanceId":     "instance-id",
		"region":         "region",
	} {
		v, ok := run.ProviderMetadata[md]
		if ok {
			run.WorkerConfig, err = run.WorkerConfig.Set(cfg, v)
			if err != nil {
				return err
			}
		} else {
			log.Printf("provider metadata %s not available; not setting config %s", md, cfg)
		}
	}

	set := func(key, value string) {
		var err error
		// only programming errors can cause this to fail
		run.WorkerConfig, err = run.WorkerConfig.Set(key, value)
		if err != nil {
			panic(err)
		}
	}

	set("rootUrl", run.RootURL)
	set("taskcluster.clientId", run.Credentials.ClientID)
	set("taskcluster.accessToken", run.Credentials.AccessToken)
	if run.Credentials.Certificate != "" {
		set("taskcluster.certificate", run.Credentials.Certificate)
	}

	set("workerId", run.WorkerID)
	set("workerGroup", run.WorkerGroup)

	workerPoolID := strings.SplitAfterN(run.WorkerPoolID, "/", 2)
	set("provisionerId", workerPoolID[0][:len(workerPoolID[0])-1])
	set("workerType", workerPoolID[1])

	return nil
}

func (d *genericworker) StartWorker(run *runner.Run) (protocol.Transport, error) {
	// write out the config file
	content, err := json.MarshalIndent(run.WorkerConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("Error constructing worker config: %v", err)
	}
	err = ioutil.WriteFile(d.wicfg.ConfigPath, content, 0600)
	if err != nil {
		return nil, fmt.Errorf("Error writing worker config to %s: %v", d.wicfg.ConfigPath, err)
	}

	// the --host taskcluster-worker-runner instructs generic-worker to merge
	// config from $GENERIC_WORKER_CONFIG.
	exe := fmt.Sprintf("%s/src/bin/worker.js", d.wicfg.Path)
	cmd := exec.Command("node", exe, "--host", "taskcluster-worker-runner", "production")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GENERIC_WORKER_CONFIG="+d.wicfg.ConfigPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	d.cmd = cmd

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return protocol.NewNullTransport(), nil
}

func (d *genericworker) SetProtocol(proto *protocol.Protocol) {
}

func (d *genericworker) Wait() error {
	return d.cmd.Wait()
}

func New(runnercfg *runner.RunnerConfig) (worker.Worker, error) {
	rv := genericworker{runnercfg, genericworkerConfig{}, nil}
	err := runnercfg.WorkerImplementation.Unpack(&rv.wicfg)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

func Usage() string {
	return `

The "generic-worker" worker implementation starts generic-worker
(https://github.com/taskcluster/generic-worker).  It takes the following
values in the 'worker' section of the runner configuration:

	worker:
		implementation: generic-worker
		# path to the root of the generic-worker repo clone
		path: /path/to/generic-worker/repo
		# path where taskcluster-worker-runner should write the generated
		# generic-worker configuration.
		configPath: ..

`
}
