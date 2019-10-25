package standalone

import (
	"github.com/taskcluster/taskcluster-worker-runner/cfg"
	"github.com/taskcluster/taskcluster-worker-runner/protocol"
	"github.com/taskcluster/taskcluster-worker-runner/provider/provider"
	"github.com/taskcluster/taskcluster-worker-runner/run"
)

type standaloneProviderConfig struct {
	RootURL      string
	ClientID     string
	AccessToken  string
	WorkerPoolID string
	WorkerGroup  string
	WorkerID     string
}

type StandaloneProvider struct {
	runnercfg *cfg.RunnerConfig
}

func setWorkerConfig(key string, data interface{}, wc *cfg.WorkerConfig) (*cfg.WorkerConfig, error) {
	var err error

	switch data.(type) {
	case map[interface{}]interface{}:
		for k, v := range data.(map[interface{}]interface{}) {
			if wc, err = setWorkerConfig(key+"."+k.(string), v, wc); err != nil {
				return nil, err
			}
		}
	case map[string]interface{}:
		for k, v := range data.(map[string]interface{}) {
			if wc, err = setWorkerConfig(key+"."+k, v, wc); err != nil {
				return nil, err
			}
		}
	default:
		wc, err = wc.Set(key, data)
	}

	return wc, err
}

func (p *StandaloneProvider) ConfigureRun(state *run.State) error {
	var pc standaloneProviderConfig
	err := p.runnercfg.Provider.Unpack(&pc)
	if err != nil {
		return err
	}

	state.RootURL = pc.RootURL
	state.Credentials.ClientID = pc.ClientID
	state.Credentials.AccessToken = pc.AccessToken
	state.WorkerPoolID = pc.WorkerPoolID
	state.WorkerGroup = pc.WorkerGroup
	state.WorkerID = pc.WorkerID
	state.WorkerLocation = map[string]string{
		"cloud": "standalone",
	}

	state.ProviderMetadata = map[string]string{}

	if workerLocation, ok := p.runnercfg.Provider.Data["workerLocation"]; ok {
		for k, v := range workerLocation.(map[string]string) {
			state.WorkerLocation[k] = v
		}
	}

	state.WorkerConfig = cfg.NewWorkerConfig()

	for k, v := range p.runnercfg.Provider.Data["userData"].(map[interface{}]interface{}) {
		state.WorkerConfig, err = setWorkerConfig(k.(string), v, state.WorkerConfig)
	}

	return err
}

func (p *StandaloneProvider) UseCachedRun(run *run.State) error {
	return nil
}

func (p *StandaloneProvider) SetProtocol(proto *protocol.Protocol) {
}

func (p *StandaloneProvider) WorkerStarted() error {
	return nil
}

func (p *StandaloneProvider) WorkerFinished() error {
	return nil
}

func New(runnercfg *cfg.RunnerConfig) (provider.Provider, error) {
	return &StandaloneProvider{runnercfg}, nil
}

func Usage() string {
	return `
The providerType "standalone" is intended for workers that have all of their
configuration pre-loaded.  Such workers do not interact with the worker manager.
This is not a recommended configuration - prefer to use the static provider.

It requires the following properties be included explicitly in the runner
configuration:

` + "```yaml" + `
provider:
    providerType: standalone
    rootURL: ..  # note the Golang spelling with capitalized "URL"
    clientID: .. # ..and similarly capitalized ID
    accessToken: ..
    workerPoolID: ..
    workerGroup: ..
    workerID: ..
    # custom properties for TASKCLUSTER_WORKER_LOCATION
    workerLocation:  {prop: val, ..}
    # custom worker configuration
	userData:
		prop: value
		...
` + "```" + `

The [$TASKCLUSTER_WORKER_LOCATION](https://docs.taskcluster.net/docs/reference/core/worker-manager/)
defined by this provider has the following fields:

* cloud: standalone

as well as any worker location values from the configuration.
`
}
