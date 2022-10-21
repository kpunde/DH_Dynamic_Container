package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"
)

type baseHandle struct{}

var dockerImage = "deephaven-snowflake_server:latest"
var userPort = make(map[string]int)
var portTrack = make(map[int]int)
var userContainer = make(map[string]string)

func getNewPort() int {
	var newPort int
	for {
		newPort = rand.Intn(60000) + 10001
		if _, ok := portTrack[newPort]; !ok {
			return newPort
		}
	}
}

func (h *baseHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client, err := getDockerClient()
	if err != nil {
		fmt.Println(err.Error())
	}

	user := r.Header.Get("x-sso-email")
	var newPort int
	if val, ok := userPort[user]; !ok {
		newPort = getNewPort()
		userPort[user] = newPort
		portTrack[newPort] = 1

		id, err := createContainerForUser(client, dockerImage, newPort, "deephaven")
		if err != nil {
			fmt.Println(err)
			return
		}
		userContainer[user] = id
		fmt.Println(id)
	} else {
		newPort = val
	}
	fmt.Println(userPort)
	fmt.Println(userContainer)

	backendURL := fmt.Sprintf("http://127.0.0.1:%v/", newPort)
	backend, _ := url.Parse(backendURL)
	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 600 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   600 * time.Second,
			KeepAlive: 600 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       900 * time.Second,
		TLSHandshakeTimeout:   100 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(err.Error())); err != nil {
			log.Println(err)
		}
	}

	proxy.ServeHTTP(w, r)
	return
}

func getDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv)
}

func getContainer(cli *client.Client) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID, container.Image)
	}
}

func createContainerForUser(cli *client.Client, image string, port int, namePrefix string) (string, error) {
	portStr := strconv.Itoa(port)
	portPStr := strconv.Itoa(port) + "/tcp"
	contName := fmt.Sprint(namePrefix + "_" + portStr)
	containerConfig := container.Config{
		Image: image,
		ExposedPorts: nat.PortSet{
			nat.Port(portStr): {},
		},
	}

	hostConfig := container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock",
		},
		PortBindings: nat.PortMap{
			nat.Port("10000/tcp"): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: portPStr,
				},
			},
		},
		Resources: container.Resources{
			Memory: 1024 * 1000000,
		},
	}

	cont, err := cli.ContainerCreate(context.Background(), &containerConfig, &hostConfig, nil, nil, contName)
	if err != nil {
		return "", err
	}
	err = cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}
	return cont.ID, nil
}

func main() {
	os.Setenv("DOCKER_API_VERSION", "1.41")
	h := &baseHandle{}
	http.Handle("/", h)

	server := &http.Server{
		Addr:    ":8090",
		Handler: h,
	}
	log.Fatal(server.ListenAndServe())
}
