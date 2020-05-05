package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.dedis.ch/simnet"
	"go.dedis.ch/simnet/network"
	"go.dedis.ch/simnet/sim"
	"go.dedis.ch/simnet/sim/kubernetes"
)

type simRound struct{}

func (s simRound) Before(simio sim.IO, nodes []sim.NodeInfo) error {
	return nil
}

func (s simRound) Execute(simio sim.IO, nodes []sim.NodeInfo) error {
	fmt.Printf("Nodes: %v\n", nodes)

	for _, node := range nodes {
		simio.Tag(node.Name)

		resp, err := http.Get(fmt.Sprintf("http://%s:80", node.Address))
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		fmt.Printf("Found page of length %d bytes for %s\n", len(body), node.Name)
		time.Sleep(time.Second)
	}

	return nil
}

func (s simRound) After(simio sim.IO, nodes []sim.NodeInfo) error {
	return nil
}

func main() {
	options := []sim.Option{
		// sim.WithTopology(
		// 	network.NewSimpleTopology(3, 25),
		// ),
		sim.WithTopology(
			network.NewCloudTopology("beta.kubernetes.io/arch", []string{"amd65"}),
		),
		sim.WithImage("nginx", nil, nil, sim.NewTCP(80)),
		// Example of a mount of type tmpfs.
		sim.WithTmpFS("/storage", 256*sim.MB),
		// Example of requesting a minimum amount of resources.
		kubernetes.WithResources("20m", "64Mi"),
	}

	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	engine, err := kubernetes.NewStrategy(kubeconfig, options...)
	// engine, err := docker.NewStrategy(options...)
	if err != nil {
		panic(err)
	}

	sim := simnet.NewSimulation(simRound{}, engine)

	err = sim.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
