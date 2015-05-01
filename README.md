# DRM - Beta
Docker Ruby Manager

This is a project I created to learn Golang. It is basically a very simple RVM type system 
that uses Docker to manage ruby containers. This system currently supports function mapping 
for rake, ruby, bundle, and rspec via the function.sh file.

## Installation

It is assumed that you have installed Docker on your machine. I have currently only tested this on Mac.

Simply run this command to install DRM on your Mac. It will setup the function mappings for you. 

```bash
docker run --rm -e USER_DIR=$HOME -v ~/:$HOME dbdevs/drm mac
```

## Commands

### install RUBY_VERSION [-r|--repo REPO_NAME]

Pulls the image for the particular Ruby version so it can be used.

Example:
```bash
drm install 2.2.2 -r dockerhub.example.com
```

This will install the image dockerhub.example.com/ruby:2.2.2.

### use RUBY_VERSION[@GEMSET] [-r|--repo REPO_NAME]

Stages the specified ruby image as a container and mounts the user's HOME directory with the 
current directory as the working directory. TODO: Change the working directory whenever `run` is used.

If no gemset is specified, then default is used.

### run

It shouldn't be necessary to call this command directly. When you do need to, please log an issue so 
I can add an alias for the command. This command can currently be accessed by `rake`, `ruby`, `rspec`, and `bundle`.

### destroy RUBY_VERSION[@GEMSET] [-r|--repo REPO_NAME]

Stops and removes the running container associated to the ruby version and gemset.

### uninstall RUBY_VERSION [-r|--repo REPO_NAME]

Removes the ruby image locally.

## Environment Variables

Input:

* DRM_DOCKER_REPO - [optional] Can be set to avoid entering it for each command.
* DOCKER_HOST - [required] This is necessary as this program communicates with the docker daemon api. 
    Can be the .sock or a tcp.
* HOME - [required] This allows for mounting a directory so you can switch directories without restarting 
    TODO: Make this more generic as it is just for Mac

Output:

These are set per shell instance and do not persist between shell instances.

* DRM_CONTAINER_NAME - The name of the current container being used.
* DRM_IMAGE_NAME - The name of the current image being used.
* DRM_FULL_IMAGE_NAME - The full name with the repository of the current image being used.
