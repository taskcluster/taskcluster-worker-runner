package awsprovisioner

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/taskcluster/httpbackoff"
)

type fakeMetadataService struct {
	UserDataError error
	UserData      *UserData
	Metadata      map[string]string
}

func (mds *fakeMetadataService) queryUserData() (*UserData, error) {
	if mds.UserDataError != nil {
		return nil, mds.UserDataError
	}
	return mds.UserData, nil
}

func (mds *fakeMetadataService) queryMetadata(path string) (string, error) {
	if path[0] != '/' {
		panic("path must start with /")
	}
	res, ok := mds.Metadata[path]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return res, nil
}

func TestQueryMetadata(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/meta-data/some-data" {
			w.WriteHeader(200)
			fmt.Fprintln(w, "42")
		} else {
			w.WriteHeader(404)
			fmt.Fprintln(w, "Not Found")
		}
	}))
	defer ts.Close()

	EC2MetadataBaseURL = ts.URL + "/latest"
	defer func() {
		EC2MetadataBaseURL = "http://169.254.169.254/latest"
	}()

	ms := realMetadataService{}

	rv, err := ms.queryMetadata("/meta-data/some-data")
	assert.NoError(t, err)
	assert.Equal(t, "42\n", rv)

	_, err = ms.queryMetadata("/meta-data/NOSUCH")
	if err != nil {
		httperr, ok := err.(httpbackoff.BadHttpResponseCode)
		assert.True(t, ok)
		assert.Equal(t, 404, httperr.HttpResponseCode)
	}
}

func TestQueryUserData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/user-data" {
			w.WriteHeader(200)
			fmt.Fprintln(w, `{"region": "aa-central-2", "custom": "custom-data", "dockerConfig": {"privileged": true}}`)
		} else {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Not Found: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	EC2MetadataBaseURL = ts.URL + "/latest"
	defer func() {
		EC2MetadataBaseURL = "http://169.254.169.254/latest"
	}()

	ms := realMetadataService{}

	ud, err := ms.queryUserData()
	assert.NoError(t, err)
	assert.Equal(t, "aa-central-2", ud.Region)
	assert.NotNil(t, ud.Data)
	assert.NotNil(t, ud.Data.Config)
	customData, err := ud.Data.Config.Get("custom")
	assert.NoError(t, err)
	assert.Equal(t, "custom-data", customData)
	privileged, err := ud.Data.Config.Get("dockerConfig.privileged")
	assert.NoError(t, err)
	assert.Equal(t, privileged, true)
	_, err = ud.Data.Config.Get("region")
	assert.Error(t, err)
}
