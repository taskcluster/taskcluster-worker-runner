package awsprovisioner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/taskcluster/taskcluster-worker-runner/cfg"
	"github.com/taskcluster/taskcluster-worker-runner/protocol"
	"github.com/taskcluster/taskcluster-worker-runner/run"
	"github.com/taskcluster/taskcluster-worker-runner/tc"
	"github.com/taskcluster/taskcluster/clients/client-go/v15/tcawsprovisioner"
)

func TestAwsProviderConfigureRun(t *testing.T) {
	runnerWorkerConfig := cfg.NewWorkerConfig()
	runnerWorkerConfig, err := runnerWorkerConfig.Set("from-user-data", false) // overridden
	assert.NoError(t, err, "setting config")
	runnerWorkerConfig, err = runnerWorkerConfig.Set("from-runner-cfg", true)
	assert.NoError(t, err, "setting config")
	runnercfg := &cfg.RunnerConfig{
		Provider: cfg.ProviderConfig{
			ProviderType: "aws-provisioner",
		},
		WorkerImplementation: cfg.WorkerImplementationConfig{
			Implementation: "whatever",
		},
		WorkerConfig: runnerWorkerConfig,
	}
	token := tc.FakeAwsProvisionerCreateSecret(&tcawsprovisioner.SecretResponse{
		Credentials: tcawsprovisioner.Credentials{
			ClientID:    "cli",
			AccessToken: "at",
			Certificate: "cert",
		},
		Data:   []byte("{}"),
		Scopes: []string{},
	})

	userDataWorkerConfig := cfg.NewWorkerConfig()
	userDataWorkerConfig, err = userDataWorkerConfig.Set("from-user-data", true)
	assert.NoError(t, err, "setting config")
	userData := &UserData{
		Data:               userDataWorkerConfig,
		WorkerType:         "wt",
		ProvisionerID:      "apv1",
		Region:             "rgn",
		TaskclusterRootURL: "https://tc.example.com",
		SecurityToken:      token,
	}

	metaData := map[string]string{
		"/meta-data/ami-id":                      "ami-123",
		"/meta-data/instance-id":                 "i-123",
		"/meta-data/instance-type":               "g12.128xlarge",
		"/meta-data/public-ipv4":                 "1.2.3.4",
		"/meta-data/placement/availability-zone": "rgna",
		"/meta-data/public-hostname":             "foo.ec2-dns",
		"/meta-data/local-ipv4":                  "192.168.0.1",
	}

	p, err := new(runnercfg, tc.FakeAwsProvisionerClientFactory, &fakeMetadataService{nil, userData, metaData})
	assert.NoError(t, err, "creating provider")

	state := run.State{
		WorkerConfig: runnercfg.WorkerConfig,
	}
	err = p.ConfigureRun(&state)
	assert.NoError(t, err, "ConfigureRun")

	assert.Nil(t, tc.FakeAwsProvisionerGetSecret(token), "secret should have been removed")

	assert.Equal(t, "https://tc.example.com", state.RootURL, "rootURL is correct")
	assert.Equal(t, "cli", state.Credentials.ClientID, "clientID is correct")
	assert.Equal(t, "at", state.Credentials.AccessToken, "accessToken is correct")
	assert.Equal(t, "cert", state.Credentials.Certificate, "cert is correct")
	assert.Equal(t, "apv1/wt", state.WorkerPoolID, "workerPoolID is correct")
	assert.Equal(t, "rgn", state.WorkerGroup, "workerGroup is correct")
	assert.Equal(t, "i-123", state.WorkerID, "workerID is correct")
	assert.Equal(t, map[string]string{
		"ami-id":            "ami-123",
		"instance-id":       "i-123",
		"instance-type":     "g12.128xlarge",
		"public-ipv4":       "1.2.3.4",
		"availability-zone": "rgna",
		"public-hostname":   "foo.ec2-dns",
		"local-ipv4":        "192.168.0.1",
		"region":            "rgn",
	}, state.ProviderMetadata, "providerMetadata is correct")

	assert.Equal(t, true, state.WorkerConfig.MustGet("from-user-data"), "value for from-user-data")
	assert.Equal(t, true, state.WorkerConfig.MustGet("from-runner-cfg"), "value for from-runner-cfg")
}

func TestCheckTerminationTime(t *testing.T) {
	transp := protocol.NewFakeTransport()
	proto := protocol.NewProtocol(transp)

	metaData := map[string]string{}
	p := &AwsProvisionerProvider{
		runnercfg:                   nil,
		awsProvisionerClientFactory: nil,
		metadataService:             &fakeMetadataService{nil, nil, metaData},
		proto:                       proto,
		terminationTicker:           nil,
	}

	p.checkTerminationTime()

	// not time yet..
	assert.Equal(t, []protocol.Message{}, transp.Messages())

	metaData["/meta-data/spot/termination-time"] = "now!"
	p.checkTerminationTime()

	// protocol does not have the capability set..
	assert.Equal(t, []protocol.Message{}, transp.Messages())

	proto.Capabilities.Add("graceful-termination")
	p.checkTerminationTime()

	// now we send a message..
	assert.Equal(t, []protocol.Message{
		protocol.Message{
			Type: "graceful-termination",
			Properties: map[string]interface{}{
				"finish-tasks": false,
			},
		},
	}, transp.Messages())
}
