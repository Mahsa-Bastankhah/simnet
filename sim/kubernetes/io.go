package kubernetes

import (
	"bytes"
	"errors"
	"io"
	"net/url"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type fs interface {
	Read(string, string, string) (io.ReadCloser, error)
	Write(string, string, []string) (io.WriteCloser, <-chan error, error)
}

type kfs struct {
	restclient   rest.Interface
	namespace    string
	makeExecutor func(*url.URL) (remotecommand.Executor, error)
}

func (fs kfs) Read(pod, container, path string) (io.ReadCloser, error) {
	reader, outStream := io.Pipe()

	req := fs.restclient.
		Get().
		Namespace(fs.namespace).
		Resource("pods").
		Name(pod).
		SubResource("exec").
		VersionedParams(&apiv1.PodExecOptions{
			Container: container,
			Command:   []string{"cat", path},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := fs.makeExecutor(req.URL())
	if err != nil {
		return nil, err
	}

	go func() {
		var err error
		defer func() {
			if err != nil {
				outStream.CloseWithError(err)
			} else {
				outStream.Close()
			}
		}()

		// Any error coming from the standard error will be written in that
		// buffer so that a better explanation can be provided to any error
		// occuring.
		outErr := new(bytes.Buffer)

		err = exec.Stream(remotecommand.StreamOptions{
			Stdout: outStream,
			Stderr: outErr,
			Tty:    false,
		})

		if err != nil {
			// Replace the generic command error message by a better message.
			err = errors.New(outErr.String())
		}
	}()

	return reader, nil
}

func (fs kfs) Write(pod, container string, cmd []string) (io.WriteCloser, <-chan error, error) {
	reader, writer := io.Pipe()

	req := fs.restclient.
		Post().
		Namespace(fs.namespace).
		Resource("pods").
		Name(pod).
		SubResource("exec").
		VersionedParams(&apiv1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := fs.makeExecutor(req.URL())
	if err != nil {
		return nil, nil, err
	}

	done := make(chan error, 1)
	out := new(bytes.Buffer)

	go func() {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  reader,
			Stdout: out,
			Stderr: out,
			Tty:    false,
		})

		done <- err
	}()

	return writer, done, nil
}
