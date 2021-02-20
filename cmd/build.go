package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types"
	builder "github.com/groenlid/docker-builder/cmd/builders"
	"github.com/groenlid/docker-builder/cmd/structs"
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

	authString, err := getRegistryAuthString(dockerusername, dockerpassword, dockerregistry)
	if err != nil {
		log.Fatalln(err)
	}
	configurations, err := findYT3ConfigurationFiles(".")

	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()

	buildAndPushImages(ctx, configurations, authString)

	persitDigestCache(digestCachePath, digestCache)
}

func runBuildOld(cmd *cobra.Command, args []string) {
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

	ctx := context.Background()

	buildAndPushImages(ctx, configurations, "")

	persitDigestCache(digestCachePath, digestCache)
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

func getRegistryAuthString(username string, password string, dockerregistry string) (string, error) {
	authConfig := types.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: dockerregistry,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

func findYT3ConfigurationFiles(sourceDirectory string) ([]structs.ConfigurationWithProjectPath, error) {
	foldersToSkip := []string{"node_modules", ".git", "bin"}
	configName := "ytbdsettings.json"
	var configs []structs.ConfigurationWithProjectPath

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

		configuration := structs.Configuration{}
		deserializeError := json.Unmarshal(fileContent, &configuration)

		if deserializeError != nil {
			log.Println(deserializeError)
			return nil
		}

		configs = append(configs, structs.ConfigurationWithProjectPath{Configuration: configuration, ProjectPath: path})
		return nil
	})

	return configs, err
}

func buildAndPushImages(ctx context.Context, configurations []structs.ConfigurationWithProjectPath, auth string) {
	for _, configuration := range configurations {
		buildDockerImage(ctx, configuration, auth)
	}
}

func buildDockerImage(ctx context.Context, configuration structs.ConfigurationWithProjectPath, auth string) {
	os.Setenv("DOCKER_BUILDKIT", "1")
	os.Setenv("BUILDKIT_PROGRESS", "plain")
	builderForProject, err := builder.Manager.GetBuilderForProject(configuration)
	if err != nil {
		log.Fatalln(err)
	}

	arguments, err := builderForProject.GetBuildArguments(configuration)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(configuration.ServiceName, builderForProject.BuilderName, arguments)
}

func pushImage() {

}

func copyDeloymentArtifactsToOutputFolder() {

}
