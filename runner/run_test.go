package runner

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/Flaque/filet"
	"github.com/stretchr/testify/assert"
)

func TestFakeGenericWorker(t *testing.T) {
	defer filet.CleanUp(t)
	dir := filet.TmpDir(t, "")
	configPath := filepath.Join(dir, "runner.yaml")

	exe := "go"
	fakeWorker := "../hack/cp.go"
	infile := filepath.Join(dir, "infile")
	outfile := filepath.Join(dir, "outfile")
	testData := "this is a test"
	err := ioutil.WriteFile(infile, []byte(testData), 0755)
	if !assert.NoError(t, err) {
		return
	}

	err = ioutil.WriteFile(configPath, []byte(fmt.Sprintf(`
provider:
  providerType: standalone
  rootURL: https://tc.example.com
  clientID: fake
  accessToken: fake
  workerPoolID: pp/ww
  workerGroup: wg
  workerID: wi
getSecrets: false
worker:
  implementation: generic-worker
  configPath: %s
  path: %s
  args:
    - run
    - '%s'
    - '%s'
    - '%s'
`, configPath, exe, fakeWorker, infile, outfile)), 0755)
	if !assert.NoError(t, err) {
		return
	}

	_, err = Run(configPath)
	if !assert.NoError(t, err) {
		return
	}

	bs, err := ioutil.ReadFile(outfile)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []byte(testData), bs)
}

func TestDummy(t *testing.T) {
	defer filet.CleanUp(t)
	dir := filet.TmpDir(t, "")
	configPath := filepath.Join(dir, "runner.yaml")

	err := ioutil.WriteFile(configPath, []byte(`
provider:
  providerType: standalone
  rootURL: https://tc.example.com
  clientID: fake
  accessToken: fake
  workerPoolID: pp/ww
  workerGroup: wg
  workerID: wi
getSecrets: false
worker:
  implementation: dummy
`), 0755)
	if !assert.NoError(t, err) {
		return
	}

	run, err := Run(configPath)
	if !assert.NoError(t, err) {
		return
	}

	// spot-check some run values; the main point here is that
	// an error does not occur
	assert.Equal(t, "https://tc.example.com", run.RootURL)
	assert.Equal(t, "fake", run.Credentials.ClientID)
	assert.Equal(t, "pp/ww", run.WorkerPoolID)
}

func TestDummyCached(t *testing.T) {
	defer filet.CleanUp(t)
	dir := filet.TmpDir(t, "")
	configPath := filepath.Join(dir, "runner.yaml")
	cachePath := filepath.Join(dir, "cache.json")

	err := ioutil.WriteFile(configPath, []byte(fmt.Sprintf(`
provider:
  providerType: standalone
  rootURL: https://tc.example.com
  clientID: fake
  accessToken: fake
  workerPoolID: pp/ww
  workerGroup: wg
  workerID: wi
getSecrets: false
cacheOverRestarts: %s
workerConfig:
  fromFirstRun: true
worker:
  implementation: dummy
`, cachePath)), 0755)
	if !assert.NoError(t, err) {
		return
	}

	run, err := Run(configPath)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, true, run.WorkerConfig.MustGet("fromFirstRun"))

	// slightly different config this time, omitting `fromFirstRun`:
	err = ioutil.WriteFile(configPath, []byte(fmt.Sprintf(`
provider:
  providerType: standalone
  rootURL: https://tc.example.com
  clientID: fake
  accessToken: fake
  workerPoolID: pp/ww
  workerGroup: wg
  workerID: wi
getSecrets: false
cacheOverRestarts: %s
worker:
  implementation: dummy
`, cachePath)), 0755)
	if !assert.NoError(t, err) {
		return
	}

	run, err = Run(configPath)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, true, run.WorkerConfig.MustGet("fromFirstRun"))
}
