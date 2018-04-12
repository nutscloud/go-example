/*
Copyright 2016 caicloud authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/builder/dockerfile/command"
	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
)

// AuthConfig contains the username and password to access docker registry.
type AuthConfig struct {
	Username string
	Password string
}

// NewAuthConfig returns a new AuthConfig or returns an error.
func NewAuthConfig(username, password string) (*AuthConfig, error) {
	if username == "" || password == "" {
		return nil, errors.New("The username or password for docker registry is not set.")
	}
	return &AuthConfig{
		Username: username,
		Password: password,
	}, nil
}

// Client manages all docker operations, like build, push, etc.
type Client struct {
	Client     *dockerclient.Client
	Registry   string
	AuthConfig *AuthConfig
	EndPoint   string
}

// RegistryCompose that compose the info about the registry
type RegistryCompose struct {
	// Registry's address, ie. cargo.caicloud.io
	RegistryLocation string `json:"registrylocation,omitempty"`
	// RegistryUsername used for operating the images
	RegistryUsername string `json:"registryusername,omitempty"`
	// RegistryPassword used for operating the images
	RegistryPassword string `json:"registrypassword,omitempty"`
}

// NewClient creates a new docker manager.
func NewClient(endpoint string, certPath string, registry *RegistryCompose) (*Client, error) {
	// Get the AuthConfig from username and password in SYSTEM ENV.
	authConfig, err := NewAuthConfig(registry.RegistryUsername, registry.RegistryPassword)
	if err != nil {
		return nil, err
	}

	if certPath == "" {
		client, err := dockerclient.NewClient(endpoint)
		if err != nil {
			return nil, err
		}

		return &Client{
			Client:     client,
			Registry:   registry.RegistryLocation,
			AuthConfig: authConfig,
			EndPoint:   endpoint,
		}, nil
	}

	cert := fmt.Sprintf("%s/cert.pem", certPath)
	key := fmt.Sprintf("%s/key.pem", certPath)
	ca := fmt.Sprintf("%s/ca.pem", certPath)

	client, err := dockerclient.NewTLSClient(endpoint, cert, key, ca)
	if err != nil {
		return nil, err
	}

	_, err = client.Version()
	if err != nil {
		glog.Errorf("error connecting to docker daemon %s. %s.", endpoint, err)
		return nil, err
	}

	return &Client{
		Client:     client,
		Registry:   registry.RegistryLocation,
		AuthConfig: authConfig,
		EndPoint:   endpoint,
	}, nil
}

// PullImage pulls an image by its name.
func (c *Client) PullImage(imageName string) error {
	repo, err := reference.Parse(imageName)
	if err != nil {
		glog.Errorf("imagename parse error: %v", err)
		return err
	}
	tagged, ok := repo.(reference.Tagged)
	if !ok || tagged.Tag() == "" {
		imageName = fmt.Sprintf("%s:%s", imageName, "latest")
	}

	opts := dockerclient.PullImageOptions{
		Repository: imageName,
		Registry:   c.Registry,
	}

	authOpt := dockerclient.AuthConfiguration{
		Username: c.AuthConfig.Username,
		Password: c.AuthConfig.Password,
	}

	err = c.Client.PullImage(opts, authOpt)
	if err == nil {
		glog.Infof("Successfully pull docker image:%s", imageName)
	}
	return err
}

// BuildImage builds image.
func (c *Client) BuildImage(imagename, tagname, contextdir string) error {
	imageName := imagename + ":" + tagname
	glog.Infof("Build docker image:%s.", imageName)

	// Use to pull cargo.caicloud.io/:username/:imagename:tag.
	// TODO: we will consider more cases
	authOpt := dockerclient.AuthConfiguration{
		Username: c.AuthConfig.Username,
		Password: c.AuthConfig.Password,
	}

	authOpts := dockerclient.AuthConfigurations{
		Configs: make(map[string]dockerclient.AuthConfiguration),
	}
	authOpts.Configs[c.Registry] = authOpt

	opt := dockerclient.BuildImageOptions{
		Name:           imageName,
		ContextDir:     contextdir,
		AuthConfigs:    authOpts,
		RmTmpContainer: true,
		Memswap:        -1,
		OutputStream:   output,
	}
	err := c.Client.BuildImage(opt)
	if err == nil {
		glog.Infof("Successfully built docker image:%s.", imageName)
	}
	return err
}

// PushImage pushes docker image to registry. output will be sent to event status output.
func (c *Client) PushImage(imageName, tagName string) error {
	glog.Infof("About to push docker image:%s:%s.", imageName, tagName)

	opt := dockerclient.PushImageOptions{
		Name:         imageName,
		Tag:          tagName,
		OutputStream: output,
	}

	authOpt := dockerclient.AuthConfiguration{
		Username: c.AuthConfig.Username,
		Password: c.AuthConfig.Password,
	}

	err := c.Client.PushImage(opt, authOpt)
	if err == nil {
		glog.Infof("Successfully pushed docker image:%s.", imageName)
	}

	return err
}

// RunContainer runs a container according to special config
func (c *Client) RunContainer(cco *dockerclient.CreateContainerOptions) (string, error) {
	isImageExisted, err := c.IsImagePresent(cco.Config.Image)
	if err != nil {
		return "", err
	}

	if isImageExisted == false {
		glog.Infof("About to pull the image:%s.", cco.Config.Image)
		err := c.PullImage(cco.Config.Image)
		if err != nil {
			return "", err
		}
		glog.Infof("Successfully pull the image:%s.", cco.Config.Image)
	}

	glog.Infof("About to create the container:%s.", *cco)
	client := c.Client
	container, err := client.CreateContainer(*cco)
	if err != nil {
		return "", err
	}

	err = client.StartContainer(container.ID, cco.HostConfig)
	if err != nil {
		client.RemoveContainer(dockerclient.RemoveContainerOptions{
			ID: container.ID,
		})
		return "", err
	}

	glog.Infof("Successfully create the container:%s.", *cco)
	return container.ID, nil
}

// StopContainer stops a container by given ID.
func (c *Client) StopContainer(ID string) error {
	return c.Client.StopContainer(ID, 0)
}

// RemoveContainer removes a container by given ID.
func (c *Client) RemoveContainer(ID string) error {
	return c.Client.RemoveContainer(dockerclient.RemoveContainerOptions{
		ID:            ID,
		RemoveVolumes: true,
		Force:         true,
	})
}

// StopAndRemoveContainer stops and removes a container by given ID.
func (c *Client) StopAndRemoveContainer(ID string) error {
	if err := c.StopContainer(ID); err != nil {
		return err
	}
	return c.RemoveContainer(ID)
}

// GetAuthOpts gets Auth options.
func (c *Client) GetAuthOpts() (authOpts dockerclient.AuthConfigurations) {
	authOpt := dockerclient.AuthConfiguration{
		Username: c.AuthConfig.Username,
		Password: c.AuthConfig.Password,
	}

	authOpts = dockerclient.AuthConfigurations{
		Configs: make(map[string]dockerclient.AuthConfiguration),
	}
	authOpts.Configs[c.Registry] = authOpt

	return authOpts
}

// BuildImageSpecifyDockerfile builds docker image with params from event with
// specify Dockerfile. Build output will be sent to event status output.
func (c *Client) BuildImageSpecifyDockerfile(imagename, tagname, contextDir string,
	dockerfileName string, output filebuffer.FileBuffer) error {

	imageName := imagename + ":" + tagname
	log.InfoWithFields("About to build docker image.", log.Fields{"image": imageName})

	// Use to pull cargo.caicloud.io/:username/:imagename:tag.
	// TODO: we will consider more cases
	authOpt := dockerclient.AuthConfiguration{
		Username: c.AuthConfig.Username,
		Password: c.AuthConfig.Password,
	}

	authOpts := dockerclient.AuthConfigurations{
		Configs: make(map[string]dockerclient.AuthConfiguration),
	}
	authOpts.Configs[c.Registry] = authOpt

	if "" == dockerfileName {
		dockerfileName = "Dockerfile"
	}
	opt := dockerclient.BuildImageOptions{
		Name:           imageName,
		Dockerfile:     dockerfileName,
		ContextDir:     contextDir,
		OutputStream:   output,
		AuthConfigs:    authOpts,
		RmTmpContainer: true,
		Memswap:        -1,
	}
	glog.Infof("Begin building image:%s", imageName)
	err := c.Client.BuildImage(opt)
	if err == nil {
		glog.Infof("Successfully built docker image:%s.", imageName)
	} else {
		glog.Errorf("Built docker image:%s failed.", imageName)
	}

	return err
}

// CleanUp removes images generated during building image
func (c *Client) CleanUp(imagename, tagname string) error {
	imageName := imagename + ":" + tagname
	glog.Infof("About to clean up docker image:%s.", imageName)

	err := c.RemoveImage(imageName)
	if err == nil {
		glog.Infof("Successfully remove docker image:%s.", imageName)
	} else {
		glog.Errorf("Remove docker image:%s failed.", imageName)
	}

	return err
}

// RemoveImage removes an image by its name or ID.
func (c *Client) RemoveImage(name string) error {
	return c.Client.RemoveImage(name)
}

// parse parses the "FROM" in the repo's Dockerfile to check the images which the build images base on
// It returns two parameters, the first one is used for recording image name, the second is used
// For storage the error inforamtion.
func parse(despath string) ([]string, error) {
	var str []string
	f, err := os.Open(despath + "/Dockerfile")
	if err != nil {
		glog.Errorf("Open dockerfile fail:%v.", err)
		return str, err
	}

	defer f.Close()
	d := parser.Directive{
		LookingForDirectives: true,
	}
	parser.SetEscapeToken(parser.DefaultEscapeToken, &d)
	nodes, _ := parser.Parse(f, &d)
	for _, node := range nodes.Children {
		if node.Value == command.From {
			if node.Next != nil {
				for n := node.Next; n != nil; n = n.Next {
					str = append(str, n.Value)
				}
			}
			break
		}
	}
	if len(str) <= 0 {
		return str, fmt.Errorf("there is no FROM")
	}
	return str, nil
}

// IsImagePresent checks if given image exists.
func (c *Client) IsImagePresent(image string) (bool, error) {
	_, err := c.Client.InspectImage(image)
	if err == nil {
		return true, nil
	}
	if err == dockerclient.ErrNoSuchImage {
		return false, nil
	}
	return false, err
}

// GetImageNameWithTag gets the image name with tag from registry, username, service name and version name.
func (c *Client) GetImageNameWithTag(username, serviceName, versionName string) string {
	return fmt.Sprintf("%s/%s/%s:%s", c.Registry, strings.ToLower(username), strings.ToLower(serviceName), versionName)
}

// GetImageNameNoTag gets the image name without tag from registry, username, service name.
func (c *Client) GetImageNameNoTag(username, serviceName string) string {
	return fmt.Sprintf("%s/%s/%s", c.Registry, strings.ToLower(username), strings.ToLower(serviceName))
}
