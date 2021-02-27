package structs

import "encoding/json"

type DotnetBuilder struct {
	Type          string `json:"type"`
	DotnetRuntime string `json:"dotnetruntime"`
}

type ManualBuilder struct {
	Type         string `json:"type"`
	DockerFile   string `json:"dockerfile"`
	BuildContext string `json:"buildcontext"`
}

type BaseBuilder struct {
	Type string `json:"type"`
}

type Configuration struct {
	ServiceName    string          `json:"servicename"`
	Cluster        string          `json:"cluster"`
	DeploymentFile string          `json:"deploymentfile"`
	Builder        json.RawMessage `json:"builder"`
}

type ConfigurationWithProjectPath struct {
	Configuration
	ProjectPath string
}
