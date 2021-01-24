package structs

type Deploy struct {
	Type         string `json:"type"`
	DockerFile   string `json:"dockerfile"`
	BuildContext string `json:"buildcontext"`
}

type Configuration struct {
	ServiceName    string `json:"servicename"`
	Cluster        string `json:"cluster"`
	DeploymentFile string `json:"deploymentfile"`
	Deploy         Deploy `json:"deploy"`
}

type ConfigurationWithProjectPath struct {
	Configuration
	ProjectPath string
}
