package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	builder "github.com/groenlid/docker-builder/cmd/builders"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds the services",
	Long:  `Builds the services under the current working directory`,
	Run: func(cmd *cobra.Command, args []string) {
		runBuild(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	buildCmd.Flags().StringP("registryUsername", "u", "", "The username for the docker registry being used")
	buildCmd.Flags().StringP("registryPassword", "p", "", "The password for the docker registry being used")
	buildCmd.Flags().StringP("registry", "r", "", "The docker registry being used")

	buildCmd.MarkFlagRequired("registryUsername")
	buildCmd.MarkFlagRequired("registryPassord")

}

func runBuild(cmd *cobra.Command, args []string) {
	fmt.Println("inside runbuild")
	digestCachePath := ".digestcache"
	digestCache := getDigestCache(digestCachePath)

	flags := cmd.Flags()
	dockerusername, _ := flags.GetString("registryUsername")
	dockerpassword, _ := flags.GetString("registryPassword")
	dockerregistry, _ := flags.GetString("registry")

	loginToAcr(dockerusername, dockerpassword, dockerregistry)

	configurations, err := findYT3ConfigurationFiles(".")

	if err != nil {
		log.Fatalln(err)
	}

	buildAndPushImages(configurations)

	fmt.Println(configurations)
	persitDigestCache(digestCachePath, digestCache)
	fmt.Println(digestCache)
}

func cleanArtifactFolders() {

}

type digestcache map[string]string

func getDigestCache(path string) digestcache {
	file, err := os.Open(path)
	cache := make(digestcache)
	if err != nil {
		log.Println(err)
		return cache
	}

	log.Printf("Found the digestcache file")

	defer file.Close()

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		return cache
	}

	deserializeError := json.Unmarshal(fileContent, &cache)

	if deserializeError != nil {
		log.Println(deserializeError)
	}
	return cache
}

func persitDigestCache(path string, digestToSave digestcache) {

	bytes, err := json.Marshal(digestToSave)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(bytes))
	writeErr := ioutil.WriteFile(path, bytes, 0666)
	if writeErr != nil {
		log.Println(writeErr)
	}
}

func loginToAcr(username string, password string, dockerregistry string) {
	log.Printf("Logging in to docker registry with username %v and password %v", username, password)
	cmd := exec.Command("docker", "login", "-u", username, "-p", password, dockerregistry)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Could not login to docker registry error: %v", err)
	}
}

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

func findYT3ConfigurationFiles(sourceDirectory string) ([]ConfigurationWithProjectPath, error) {
	foldersToSkip := []string{"node_modules", ".git", "bin"}
	configName := "ytbdsettings.json"
	configs := []ConfigurationWithProjectPath{}

	err := filepath.Walk(sourceDirectory, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		if info.IsDir() {
			for _, folderToSkip := range foldersToSkip {
				if folderToSkip == info.Name() {
					return filepath.SkipDir
				}
			}
		}

		if !info.Mode().IsRegular() || info.Name() != configName {
			return nil
		}

		log.Printf("Found configuration file at path: %v", path)

		fileContent, err := ioutil.ReadFile(path)
		if err != nil {
			log.Println(err)
			return nil
		}

		configuration := Configuration{}
		deserializeError := json.Unmarshal(fileContent, &configuration)

		if deserializeError != nil {
			log.Println(deserializeError)
			return nil
		}

		configs = append(configs, ConfigurationWithProjectPath{Configuration: configuration, ProjectPath: path})
		return nil
	})

	return configs, err
}

func buildAndPushImages(configurations []ConfigurationWithProjectPath) {
	for _, configuration := range configurations {
		buildDockerImage(configuration)
	}
}

func buildDockerImage(configuration ConfigurationWithProjectPath) {
	os.Setenv("DOCKER_BUILDKIT", "1")
	os.Setenv("BUILDKIT_PROGRESS", "plain")
	builder.Manager.PrintBuilders()
}

func pushImage() {

}

func copyDeloymentArtifactsToOutputFolder() {

}
