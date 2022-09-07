package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"go.dedis.ch/simnet"
	"go.dedis.ch/simnet/network"
	"go.dedis.ch/simnet/sim"
	"go.dedis.ch/simnet/sim/docker"
	"golang.org/x/xerrors"
)

type dkgSimple struct{}

const n = 4

func (s dkgSimple) Before(simio sim.IO, nodes []sim.NodeInfo) error {
	return nil
}

func (s dkgSimple) Execute(simio sim.IO, nodes []sim.NodeInfo) error {
	fmt.Printf("Nodes: %v\n", nodes)

	// initiating the log file for writing the delay and throughput data
	f, err := os.OpenFile("../../logs/test.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return xerrors.Errorf("failed to open a log file: %v", err)
	}
	defer f.Close()
	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

	out := &bytes.Buffer{}
	outErr := &bytes.Buffer{}
	opts := sim.ExecOptions{
		Stdout: out,
		Stderr: outErr,
	}

	// 1. Exchange certificates
	args := []string{"dkgcli", "--config", "/config", "minogrpc", "token"}

	err = simio.Exec(nodes[0].Name, args, opts)
	if err != nil {
		fmt.Printf("Cert exchange crashed: %v %v", out, outErr)
		return xerrors.Errorf("failed to exec cmd: %v", err)
	}

	connStr := strings.Trim(out.String(), " \n\r")

	fmt.Printf("1[%s] - Token: %q\n", nodes[0].Name, out.String())

	args = append([]string{"dkgcli", "--config", "/config", "minogrpc", "join",
		"--address", "//" + nodes[0].Address + ":2000"}, strings.Split(connStr, " ")...)

	for i := 1; i < len(nodes); i++ {
		out.Reset()
		err = simio.Exec(nodes[i].Name, args, opts)
		if err != nil {
			return xerrors.Errorf("failed to join: %v", err)
		}

		fmt.Printf("2[%s] - Join: %q\n", nodes[i].Name, out.String())
		time.Sleep(2 * time.Second)
	}

	// 2. DKG listen
	args = []string{"dkgcli", "--config", "/config", "dkg", "listen"}

	for _, node := range nodes {
		out.Reset()
		err = simio.Exec(node.Name, args, opts)
		if err != nil {
			return xerrors.Errorf("failed to listen: %v", err)
		}

		fmt.Printf("3[%s] - Listen: %q\n", node.Name, out.String())
	}

	// 3. DKG setup
	authorities := make([]string, len(nodes)*2)
	for i, node := range nodes {
		rc, err := simio.Read(node.Name, "/config/dkgauthority")
		if err != nil {
			return xerrors.Errorf("failed to read dkgauthority file: %v", err)
		}

		authority, err := ioutil.ReadAll(rc)
		if err != nil {
			return xerrors.Errorf("failed to read authority: %v", err)
		}

		authorities[i*2] = "--authority"
		authorities[i*2+1] = string(authority)

		//fmt.Printf("4[%s] - Read authority: %q\n", node.Name, string(authority))
	}

	args = append([]string{"dkgcli", "--config", "/config", "dkg", "setup", "--threshold"}, fmt.Sprint(n))
	args = append(args, authorities...)
	out.Reset()

	start := time.Now()
	err = simio.Exec(nodes[0].Name, args, opts)
	if err != nil {
		return xerrors.Errorf("failed to setup: %v", err)
	}
	setupTime := time.Since(start)

	fmt.Printf("5[%s] - Setup: %q\n", nodes[0].Name, out.String())

	// 4. verifiable Encrypt
	var messages [][]byte
	var ciphertexts string

	batchSizeSlice := []int{1, 2, 4, 8, 16}

	for _, batchSize := range batchSizeSlice {
		for i := 0; i < batchSize; i++ {

			message := make([]byte, 20)

			_, err = rand.Read(message)
			if err != nil {
				return xerrors.Errorf("failed to generate random message: %v", err)
			}

			messages = append(messages, message)

			args = append([]string{"dkgcli", "--config", "/config", "dkg", "verifiableEncrypt", "--GBar"},
				"1d0194fdc2fa2ffcc041d3ff12045b73c86e4ff95ff662a5eee82abdf44a53c7", "--message", hex.EncodeToString(message))

			out.Reset()

			err = simio.Exec(nodes[1].Name, args, opts)
			if err != nil {
				return xerrors.Errorf("failed to call encrypt: %v", err)
			}

			ciphertext := strings.Trim(out.String(), " \n\r")
			ciphertexts = ciphertexts + ciphertext

			//fmt.Printf("6[%s] - Encrypt: %q\n", nodes[1].Name, ciphertext)
			fmt.Printf("6[%s] - Encrypt %d\n", nodes[1].Name, i)

		}

		// 5. Decrypt
		args = append([]string{"dkgcli", "--config", "/config", "dkg", "verifiableDecrypt", "--ciphertexts"}, strings.TrimSuffix(ciphertexts, ":"),
			"--GBar", "1d0194fdc2fa2ffcc041d3ff12045b73c86e4ff95ff662a5eee82abdf44a53c7")
		out.Reset()

		start = time.Now()
		err = simio.Exec(nodes[2].Name, args, opts)
		if err != nil {
			return xerrors.Errorf("failed to call decrypt: %v", err)
		}
		decryptionTime := time.Since(start)

		decrypted := strings.Trim(out.String(), " \n\r")

		fmt.Printf("7[%s] - Decrypt: %q\n", nodes[2].Name, decrypted)

		// 6. Assert
		fmt.Printf("📄 Original message (hex):\t%x\n🔓 Decrypted message (hex):\t%s", messages, decrypted)
		fmt.Println()

		log.Printf("n = %d , batchSize = %d  ,decryption time = %v s, throughput =  %v tx/s , dkg setup time = %v s",
			n, batchSize, decryptionTime.Seconds(), float32(batchSize)/float32(decryptionTime.Seconds()), float32(setupTime.Seconds()))

	}

	return nil
}

func (s dkgSimple) After(simio sim.IO, nodes []sim.NodeInfo) error {
	return nil
}

func main() {
	startArgs := []string{"--config", "/config", "start",
		"--routing", "tree", "--listen", "tcp://0.0.0.0:2000"}

	options := []sim.Option{
		sim.WithTopology(
			network.NewSimpleTopology(n, time.Millisecond*10),
		),
		sim.WithImage("bastankhah/f3b:latest5", []string{}, []string{}, sim.NewTCP(2000)),
		sim.WithUpdate(func(opts *sim.Options, _, IP string) {
			opts.Args = append(startArgs, "--public", fmt.Sprintf("//%s:2000", IP))
		}),
	}

	// configuration for Kubernetes
	//kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	//engine, err := kubernetes.NewStrategy(kubeconfig, options...)

	// configuration for minikube
	engine, err := docker.NewStrategy(options...)
	if err != nil {
		panic(err)
	}

	sim := simnet.NewSimulation(dkgSimple{}, engine)

	err = sim.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
