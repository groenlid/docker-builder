package structs

type BuilderConfig struct {
	Type         string `json:"type"`
	DockerFile   string `json:"dockerfile"`
	BuildContext string `json:"buildcontext"`
}

type Configuration struct {
	ServiceName    string        `json:"servicename"`
	Cluster        string        `json:"cluster"`
	DeploymentFile string        `json:"deploymentfile"`
	Builder        BuilderConfig `json:"builder"`
}

type ConfigurationWithProjectPath struct {
	Configuration
	ProjectPath string
}
