package main

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"bufio"
	"log"
	"os/exec"
	"strings"
	"net/http"
	"encoding/json"
	"github.com/codegangsta/cli"
)

var repo = "default"
type rubyConfig struct {
	version, gemset string
}

type containerConfig struct {
	imageName, containerName, fullImageName string
}

func main() {
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
	}

	app.Run(os.Args)
}

func use(c  *cli.Context) {
	ruby := new(rubyConfig)
	container := new(containerConfig)
	println("Ruby version: ", c.Args().First())
	firstArg := c.Args().First()
	println(firstArg)
	if strings.Contains(firstArg, "@") {
		rubyParsed := strings.Split(firstArg, "@")
		ruby.version = rubyParsed[0]
		ruby.gemset = rubyParsed[1]
	} else {
		ruby.version = firstArg
		ruby.gemset = "default"
	}
	println("Ruby Version: ", ruby.version)
	println("Ruby Gemset: ", ruby.gemset)
	versionExistsLocally := versionExistsLocally(ruby.version)
	var imageName string
	if ruby.version == "default" {
		imageName = "ruby"
	} else {
		if !versionExistsLocally {
			if !versionExistsRemotely(ruby.version) {
				log.Fatal("Version does not exist")
			}
		}

		imageName = fmt.Sprintf("ruby:%s", ruby.version)
	}

	if !versionExistsLocally {
		fmt.Printf("Retrieving version [%s] from repository...\n", ruby.version)
		command := exec.Command("docker", "pull", imageName)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Run()
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

	command := exec.Command("docker", "exec", "-it", container.containerName, "/bin/bash")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	command.Run()
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

func versionExistsLocally(version string) bool {
	var versionExists bool
	cmd := exec.Command("docker", "images")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("error occured")
		fmt.Printf("%s", err)
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if strings.Contains(fields[0], "ruby") && strings.Contains(fields[1], version) {
				versionExists = true
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		os.Exit(1)
	}

	return versionExists
}

func rubyAlreadyRunning(containerName string) bool {
	out, err := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName)).Output()
	if err != nil {
		log.Fatal(err)
	}

	exists := strings.Contains(string(out), containerName)

	out, err = exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName)).Output()
	if err != nil {
		log.Fatal(err)
	}

	running := strings.Contains(string(out), containerName)

	if exists && !running {
		out, err = exec.Command("docker", "start", containerName).Output()
		if err != nil {
			log.Fatal(err)
		}

		out, err = exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName)).Output()
		if err != nil {
			log.Fatal(err)
		}

		running = strings.Contains(string(out), containerName)
	}

	if exists && !running {
		out, err = exec.Command("docker", "rm", "-f", containerName).Output()
		if err != nil {
			log.Fatal(err)
		}
	}

	return exists && running
}

func stageRubyInstance(ruby *rubyConfig, imageName string, container *containerConfig) {
	cmd := "docker"
	var cmdArgs []string

	cmdArgs = append(cmdArgs, "run")
	cmdArgs = append(cmdArgs, "-d")
	cmdArgs = append(cmdArgs, "--name")
	cmdArgs = append(cmdArgs, container.containerName)

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	home := os.Getenv("HOME")
	cmdArgs = append(cmdArgs, "-v")
	cmdArgs = append(cmdArgs, fmt.Sprintf("%s:%s", home, home))
	cmdArgs = append(cmdArgs, "-w")
	cmdArgs = append(cmdArgs, dir)
	cmdArgs = append(cmdArgs, container.fullImageName)
	cmdArgs = append(cmdArgs, "sleep")
	cmdArgs = append(cmdArgs, "infinity")

	fmt.Printf("%s", cmdArgs)

	command := exec.Command(cmd, cmdArgs...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Run()
}

func run(c  *cli.Context) {

}
