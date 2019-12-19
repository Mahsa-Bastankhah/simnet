package kubernetes

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	restfake "k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/remotecommand"
)

func TestIO_Read(t *testing.T) {
	newExecutor = func(cfg *rest.Config, m string, u *url.URL) (remotecommand.Executor, error) {
		return testExecutorFactory(nil, nil, u)
	}

	fs := kfs{
		restclient: &restfake.RESTClient{},
	}

	reader, err := fs.Read("pod", "container", "this/is/a/file")
	require.NoError(t, err)

	data, err := ioutil.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, []byte("deadbeef"), data)
}

func TestIO_ReadFailure(t *testing.T) {
	e := errors.New("make executor error")
	newExecutor = func(cfg *rest.Config, m string, u *url.URL) (remotecommand.Executor, error) {
		return testExecutorFactory(e, nil, u)
	}

	fs := kfs{
		restclient: &restfake.RESTClient{},
	}

	_, err := fs.Read("", "", "")
	require.Error(t, err)
	require.Equal(t, e, err)

	newExecutor = func(cfg *rest.Config, m string, u *url.URL) (remotecommand.Executor, error) {
		return testFailingExecutorFactory(u)
	}
	reader, err := fs.Read("", "", "")
	require.NoError(t, err)

	_, err = reader.Read(make([]byte, 1))
	require.Error(t, err)
	require.Equal(t, "stream error", err.Error())
}

func TestIO_Write(t *testing.T) {
	out := new(bytes.Buffer)
	newExecutor = func(cfg *rest.Config, m string, u *url.URL) (remotecommand.Executor, error) {
		return testExecutorFactory(nil, out, u)
	}

	fs := kfs{
		restclient: &restfake.RESTClient{},
	}

	w, done, err := fs.Write("", "", []string{})
	require.NoError(t, err)
	require.NotNil(t, w)

	content := []byte("deadbeef")
	w.Write(content)
	w.Close()
	err = <-done
	require.NoError(t, err)
	require.Equal(t, content, out.Bytes())
}

func TestIO_WriteFailure(t *testing.T) {
	e := errors.New("make executor error")
	newExecutor = func(cfg *rest.Config, m string, u *url.URL) (remotecommand.Executor, error) {
		return testExecutorFactory(e, nil, u)
	}

	fs := kfs{
		restclient: &restfake.RESTClient{},
	}

	_, _, err := fs.Write("", "", []string{})
	require.Error(t, err)
	require.Equal(t, e, err)

	newExecutor = func(cfg *rest.Config, m string, u *url.URL) (remotecommand.Executor, error) {
		return testFailingExecutorFactory(u)
	}

	_, done, err := fs.Write("", "", []string{})
	require.NoError(t, err)

	err = <-done
	require.Error(t, err)
	require.Equal(t, "command failed: stream error", err.Error())
}

type fakeExecutor struct {
	url *url.URL
	out *bytes.Buffer
	err error
}

func testExecutorFactory(err error, out *bytes.Buffer, u *url.URL) (remotecommand.Executor, error) {
	if err != nil {
		return nil, err
	}

	return &fakeExecutor{u, out, nil}, nil
}

func testFailingExecutorFactory(u *url.URL) (remotecommand.Executor, error) {
	return &fakeExecutor{u, nil, errors.New("stream error")}, nil
}

func (e *fakeExecutor) Stream(options remotecommand.StreamOptions) error {
	if e.err != nil {
		options.Stderr.Write([]byte(e.err.Error()))
		return errors.New("command failed")
	}

	if e.out != nil {
		v, err := ioutil.ReadAll(options.Stdin)
		if err != nil {
			return err
		}

		e.out.Write(v)
		if err != nil {
			return err
		}
	} else if options.Stdout != nil {
		_, err := options.Stdout.Write([]byte("deadbeef"))
		return err
	}

	return nil
}
