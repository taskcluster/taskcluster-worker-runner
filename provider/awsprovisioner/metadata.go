package awsprovisioner

import (
	"encoding/json"
	"io/ioutil"

	"github.com/taskcluster/httpbackoff"
	"github.com/taskcluster/taskcluster-worker-runner/cfg"
)

var EC2MetadataBaseURL = "http://169.254.169.254/latest"
var userDataPredefinedFields = []string{
	"data",
	"workerType",
	"provisionerId",
	"region",
	"taskclusterRootUrl",
	"securityToken",
	"capacity",
}

type userDataData struct {
	Config *cfg.WorkerConfig `json:"config"`
}

// taken from https://github.com/taskcluster/aws-provisioner/blob/5a2bc7c57b20df00f9c4357e0daeb7967e6f5ee8/lib/worker-type.js#L607-L624
type UserData struct {
	Data               userDataData `json:"data"`
	WorkerType         string       `json:"workerType"`
	ProvisionerID      string       `json:"provisionerId"`
	Region             string       `json:"region"`
	TaskclusterRootURL string       `json:"taskclusterRootUrl"`
	SecurityToken      string       `json:"securityToken"`
	Capacity           int          `json:"capacity"`
}

type MetadataService interface {
	// Query the UserData and return the parsed contents
	queryUserData() (*UserData, error)

	// Query an aribtrary metadata value; path is the portion following `latest`
	queryMetadata(path string) (string, error)
}

type realMetadataService struct{}

func (mds *realMetadataService) queryUserData() (*UserData, error) {
	// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html#instancedata-user-data-retrieval
	content, err := mds.queryMetadata("/user-data")
	if err != nil {
		return nil, err
	}
	userData := &UserData{}
	if err = json.Unmarshal([]byte(content), userData); err != nil {
		return nil, err
	}

	if userData.Data.Config == nil {
		userData.Data.Config = cfg.NewWorkerConfig()
	}

	customData := make(map[string]interface{})
	if err = json.Unmarshal([]byte(content), &customData); err != nil {
		return nil, err
	}

	// docker-worker expects any custom user data config to be passed directly to it
out:
	for k, v := range customData {
		// Do not add pre-defined fields
		for _, d := range userDataPredefinedFields {
			if k == d {
				continue out
			}
		}
		if userData.Data.Config, err = userData.Data.Config.Set(k, v); err != nil {
			return nil, err
		}
	}

	return userData, err
}

func (mds *realMetadataService) queryMetadata(path string) (string, error) {
	// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html#instancedata-data-retrieval
	// call http://169.254.169.254/latest/meta-data/instance-id with httpbackoff
	resp, _, err := httpbackoff.Get(EC2MetadataBaseURL + path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	return string(content), err
}
