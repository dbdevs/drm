package main

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"net/http"
	"encoding/json"
	"crypto/x509"
	"github.com/barkerd427/dockerclient"
	"github.com/codegangsta/cli"
	"crypto/tls"
)

var repo = "default"
type rubyConfig struct {
	version, gemset string
}

type containerConfig struct {
	imageName, containerName, fullImageName string
}

var docker *dockerclient.DockerClient

func main() {
	cert, err := tls.LoadX509KeyPair("/Users/db030112/.boot2docker/certs/boot2docker-vm/cert.pem", "/Users/db030112/.boot2docker/certs/boot2docker-vm/key.pem")
	if err != nil {
		log.Fatal(err)
	}

	caCert, err := ioutil.ReadFile("/Users/db030112/.boot2docker/certs/boot2docker-vm/ca.pem")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()

	docker, _ = dockerclient.NewDockerClient("tcp://127.0.0.1:2376", tlsConfig)

	app := cli.NewApp()
	app.Name = "Docker Ruby Manager"
	app.Usage = "Manage your ruby versions with Docker"
	app.Version = "0.1.0"
	app.Author = "Daniel Barker"

	app.Commands = []cli.Command{
		{
			Name:      "use",
			Usage:     "Selects the Ruby version to use",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:    "repo, r",
					Usage:  "The Docker repository to use",
				},
			},
			Action: use,
		},
		{
			Name:	"run",
			Usage:	"Run any ruby type command",
			Action:	run,
		},
		{
			Name:	"install",
			Usage:	"Install Ruby versions",
			Action:	install,
		},
	}

	app.Run(os.Args)
}

func install(c *cli.Context) {
	ruby := new(rubyConfig)
	firstArg := c.Args().First()

	if strings.Contains(firstArg, "@") {
		rubyParsed := strings.Split(firstArg, "@")
		ruby.version = rubyParsed[0]
		ruby.gemset = rubyParsed[1]
	} else {
		ruby.version = firstArg
		ruby.gemset = "default"
	}

	var imageName string
	if ruby.version == "default" {
		imageName = "ruby"
	} else {
		imageName = fmt.Sprintf("ruby:%s", ruby.version)
	}

	existsLocally := versionExistsLocally(imageName)

	if !existsLocally {
		if !versionExistsRemotely(ruby.version) {
			log.Fatal("Version does not exist")
		}
	}

	if !existsLocally {
		fmt.Printf("Retrieving version [%s] from repository...\n", ruby.version)
		docker.PullImage(imageName, nil)
	}

	if versionExistsLocally(imageName) {
		fmt.Printf("Ruby [%s] is ready to use!", imageName)
	} else {
		fmt.Printf("Ruby [%s] was not successfully installed. Please try downloading it manually using `docker pull %s`", imageName, imageName)
	}
}

func use(c  *cli.Context) {
	ruby := new(rubyConfig)
	container := new(containerConfig)
	firstArg := c.Args().First()

	if strings.Contains(firstArg, "@") {
		rubyParsed := strings.Split(firstArg, "@")
		ruby.version = rubyParsed[0]
		ruby.gemset = rubyParsed[1]
	} else {
		ruby.version = firstArg
		ruby.gemset = "default"
	}

	var imageName string
	if ruby.version == "default" {
		imageName = "ruby"
	} else {
		imageName = fmt.Sprintf("ruby:%s", ruby.version)
	}

	versionExistsLocally := versionExistsLocally(imageName)
	if !versionExistsLocally {
		log.Fatal("Version is not available locally. You must install it first.")
	}

	container.imageName = imageName
	container.containerName = fmt.Sprintf("drm_%s_%s", ruby.version, ruby.gemset)
	if repo != "default" {
		container.fullImageName = fmt.Sprintf("%s/%s", repo, imageName)
	} else {
		container.fullImageName = fmt.Sprintf("%s", imageName)
	}

	rubyAlreadyRunning := rubyAlreadyRunning(container.containerName)

	if !rubyAlreadyRunning {
		stageRubyInstance(ruby, imageName, container)
	}

	fmt.Printf("DRM_CONTAINER_NAME=%s\n", container.containerName)
	fmt.Printf("DRM_IMAGE_NAME=%s\n", container.imageName)
	fmt.Printf("DRM_FULL_IMAGE_NAME=%s\n", container.fullImageName)
}

func versionExistsRemotely(version string) bool {
	var versionExists bool

	resp, err := http.Get("https://index.docker.io/v1/repositories/ruby/tags")
	if err != nil {
		fmt.Println("error occured")
		fmt.Printf("%s", err)
	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	for {
		var dat []map[string]interface{}
		if err := decoder.Decode(&dat); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		for _, image := range dat {
			if version == image["name"].(string) {
				versionExists = true
			}
		}
	}

	return versionExists
}

func versionExistsLocally(imageName string) bool {
	var versionExists bool
	images, _ := docker.ListImages()
	for _, image := range images {
		for _, name := range image.RepoTags {
			if name == imageName {
				versionExists = true
			}
		}
	}

	return versionExists
}

func rubyAlreadyRunning(containerName string) bool {
	names := []string{containerName}
	filters := map[string][]string{"name": names}

	filter, _ := json.Marshal(filters)
	containers, _ := docker.ListContainers(true, false, string(filter))
	exists := len(containers) >= 1

	filters["status"] = []string{"running"}
	filter, _ = json.Marshal(filters)
	containers, _ = docker.ListContainers(true, false, string(filter))
	running := len(containers) >= 1

	if exists && !running {
		docker.StartContainer(containerName, nil)

		containers, _ = docker.ListContainers(true, false, string(filter))
		running = len(containers) >= 1
	}

	if exists && !running {
		docker.RemoveContainer(containerName, true, false)
	}

	return exists && running
}

func stageRubyInstance(ruby *rubyConfig, imageName string, container *containerConfig) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	home := os.Getenv("HOME")

	config := &dockerclient.ContainerConfig{
		Cmd: []string{"sleep", "infinity"},
		AttachStdin: false,
		AttachStderr: false,
		AttachStdout: false,
		Tty: false,
		WorkingDir: dir,
		Image: container.fullImageName,
		HostConfig: dockerclient.HostConfig{
			Binds: []string{ fmt.Sprintf("%s:%s", home, home) },
			NetworkMode: "bridge",
		},
	}

	docker.CreateContainer(config, container.containerName)
	docker.StartContainer(container.containerName, nil)
}

func run(c  *cli.Context) {
	container := new(containerConfig)

	container.containerName = os.Getenv("DRM_CONTAINER_NAME")
	container.imageName = os.Getenv("DRM_IMAGE_NAME")
	container.fullImageName = os.Getenv("DRM_FULL_IMAGE_NAME")
	log.Printf("Container name: %s\n", container.containerName)

	if container.containerName == "" {
		log.Fatalf("You have to run the command `drm use %s` prior to using the `run` command for this ruby %s", container.imageName, container.containerName)
	}

	if !rubyAlreadyRunning(container.containerName) {
		log.Fatalf("The ruby is not setup properly. Please run the `drm use %s` command for this ruby %s", container.imageName, container.containerName)
	}

	command := c.Args()
	log.Printf("Command: %s\n", command)
	execConfig := &dockerclient.ExecConfig{
		AttachStdin:	true,
		AttachStdout:	true,
		AttachStderr:	true,
		Tty:			true,
		Container:		container.containerName,
		Cmd:			command,
		Detach:			false,
	}
	output, _ := docker.Exec(execConfig)
	log.Print(output)
}
