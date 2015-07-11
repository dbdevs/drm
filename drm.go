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

type rubyConfig struct {
	version, gemset string
}

type containerConfig struct {
	imageName, containerName, fullImageName string
}

var docker *dockerclient.DockerClient

func main() {
	
	home := os.Getenv("HOME")
	certFile := fmt.Sprintf("%s/.boot2docker/certs/boot2docker-vm/cert.pem", home)
	keyFile := fmt.Sprintf("%s/.boot2docker/certs/boot2docker-vm/key.pem", home)
	caFile := fmt.Sprintf("%s/.boot2docker/certs/boot2docker-vm/ca.pem", home)
	
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	caCert, err := ioutil.ReadFile(caFile)
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
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		log.Fatal("Please set your DOCKER_HOST environment variable.")
	}
	docker, _ = dockerclient.NewDockerClient(host, tlsConfig)

	app := cli.NewApp()
	app.Name = "Docker Ruby Manager"
	app.Usage = "Manage your ruby versions with Docker"
	app.Version = "0.1.0"
	app.Author = "Daniel Barker"

	app.Commands = []cli.Command{
		{
			Name:	"install",
			Usage:	"Install Ruby versions",
			Action:	install,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:	"repo, r",
					Usage:	"The Docker repository to use",
					EnvVar:	"DRM_DOCKER_REPO",
				},
			},
		},
		{
			Name:      "use",
			Usage:     "Selects the Ruby version to use",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:    "repo, r",
					Usage:  "The Docker repository to use",
					EnvVar:	"DRM_DOCKER_REPO",
				},
			},
			Action: use,
		},
		{
			Name:	"run",
			Usage:	"Run any ruby type command",
			Action:	run,
			SkipFlagParsing: true,
		},
		{
			Name:	"destroy",
			Usage:	"Destroy a running ruby and gemset",
			Action:	destroy,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:	"repo, r",
					Usage:	"The Docker repository to use",
					EnvVar:	"DRM_DOCKER_REPO",
				},
			},
		},
		{
			Name:	"uninstall",
			Usage:	"Uninstall an installed ruby",
			Action: uninstall,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:	"repo, r",
					Usage:	"The Docker repository to use",
					EnvVar:	"DRM_DOCKER_REPO",
				},
			},
		},
	}

	app.Run(os.Args)
}

func install(c *cli.Context) {
	repo := c.String("repo")
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

	var fullImageName string
	if repo != "" {
		fullImageName = fmt.Sprintf("%s/%s", repo, imageName)
	} else {
		fullImageName = fmt.Sprintf("%s", imageName)
	}

	existsLocally := versionExistsLocally(fullImageName)

	if !existsLocally {
		var repoForRemoteCall string
		if repo == "" {
			repoForRemoteCall = "index.docker.io"
		} else {
			repoForRemoteCall = repo
		}
		if !versionExistsRemotely(repoForRemoteCall, ruby.version) {
			log.Fatal("Version does not exist")
		}
	}

	if !existsLocally {
		fmt.Printf("Retrieving [%s] from repository...\n", fullImageName)
		docker.PullImage(fullImageName, nil)
	}

	if versionExistsLocally(fullImageName) {
		fmt.Printf("Ruby [%s] is ready to use!", fullImageName)
	} else {
		fmt.Printf("Ruby [%s] was not successfully installed. Please try downloading it manually using `docker pull %s`", ruby.version, fullImageName)
	}
}

func use(c  *cli.Context) {
	repo := c.String("repo")
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
	if repo == "" {
		container.containerName = fmt.Sprintf("drm_%s_%s", ruby.version, ruby.gemset)
	} else {
		container.containerName = fmt.Sprintf("drm_%s_%s_%s", ruby.version, ruby.gemset, repo)
	}

	if repo != "" {
		container.fullImageName = fmt.Sprintf("%s/%s", repo, imageName)
	} else {
		container.fullImageName = fmt.Sprintf("%s", imageName)
	}

	rubyAlreadyRunning := rubyAlreadyRunning(container)

	if !rubyAlreadyRunning {
		stageRubyInstance(ruby, container)
	}

	fmt.Printf("DRM_CONTAINER_NAME=%s\n", container.containerName)
	fmt.Printf("DRM_IMAGE_NAME=%s\n", container.imageName)
	fmt.Printf("DRM_FULL_IMAGE_NAME=%s\n", container.fullImageName)
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

	if !rubyAlreadyRunning(container) {
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

func destroy(c *cli.Context) {
	repo := c.String("repo")
	container := new(containerConfig)

	if len(c.Args()) == 0 {
		container.containerName = os.Getenv("DRM_CONTAINER_NAME")
	} else {
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
if repo == "" {
	container.containerName = fmt.Sprintf("drm_%s_%s", ruby.version, ruby.gemset)
} else {
	container.containerName = fmt.Sprintf("drm_%s_%s_%s", ruby.version, ruby.gemset, repo)
}

	}

	docker.RemoveContainer(container.containerName, true, false)
}

func uninstall(c *cli.Context) {
	repo := c.String("repo")
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

	var fullImageName string
	if repo != "" {
		fullImageName = fmt.Sprintf("%s/%s", repo, imageName)
	} else {
		fullImageName = fmt.Sprintf("%s", imageName)
	}

	docker.RemoveImage(fullImageName)
}

func versionExistsRemotely(repo, version string) bool {
	var versionExists bool
	repoIsV2 := true
	url := fmt.Sprintf("http://%s/v2/ruby/tags/list", repo)
	resp, err := http.Get(url)
	if err != nil {
		repoIsV2 = false
		fmt.Printf("%s", err)
		url = fmt.Sprintf("http://%s/v1/repositories/ruby/tags", repo)
		resp, err = http.Get(url)
		if err != nil {
			log.Fatal("Error connecting to v1 and v2 repositories.")
		}
	}

	if repoIsV2 {
		defer resp.Body.Close()
		contents, _ := ioutil.ReadAll(resp.Body)
		decoder := json.NewDecoder(strings.NewReader(string(contents)))
		for {
			var dat struct {
				Name string
				Tags []string
			}
			if err := decoder.Decode(&dat); err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}

			for _, tag := range dat.Tags {
				if version == tag {
					return true
				}
			}
		}
	} else {
		defer resp.Body.Close()
		contents, _ := ioutil.ReadAll(resp.Body)
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

func rubyAlreadyRunning(container *containerConfig) bool {
	var exists, running bool
	names := []string{container.containerName}
	filters := map[string][]string{"name": names}

	filter, _ := json.Marshal(filters)
	containers, _ := docker.ListContainers(true, false, string(filter))

	for _, returnedContainer := range containers {
		if returnedContainer.Image == container.fullImageName {
			exists = true
			if strings.Contains(returnedContainer.Status, "Up") {
				return true
			}
		}
	}

	if exists && !running {
		docker.StartContainer(container.containerName, nil)

		containers, _ = docker.ListContainers(true, false, string(filter))
		for _, returnedContainer := range containers {
			if strings.Contains(returnedContainer.Status, "Up") {
				return true
			}
		}
	}

	if exists && !running {
		docker.RemoveContainer(container.containerName, true, false)
	}

	return exists && running
}

func stageRubyInstance(ruby *rubyConfig, container *containerConfig) {
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
