package generate

import (
	"fmt"
	"net/http"
	"os"

	shell "github.com/ipfs/go-ipfs-api"
)

func (g *Generate) newIpfsClient() (*shell.Shell, error) {
	if g.config.IPFS == nil {
		return nil, fmt.Errorf("Missing IPFS configuration")
	}

	var client *http.Client
	if g.config.IPFS.ProjectID != "" && g.config.IPFS.ProjectSecret != "" {
		client = &http.Client{
			Transport: authTransport{
				RoundTripper:  http.DefaultTransport,
				ProjectId:     g.config.IPFS.ProjectID,
				ProjectSecret: g.config.IPFS.ProjectSecret,
			},
		}
	}
	shell := shell.NewShellWithClient(g.config.IPFS.Endpoint, client)

	return shell, nil
}

func (g *Generate) Upload(path string) (string, error) {
	var cid string
	//Get path info
	fileInfo, err := os.Stat(path)
	if err != nil {
		return cid, err
	}

	//Use AddDir if the path is a directory
	if fileInfo.IsDir() {
		cid, err = g.ipfs.AddDir(path)
		if err != nil {
			return cid, err
		}
	} else { //Upload single file otherwise
		file, err := os.Open(path)
		if err != nil {
			return cid, err
		}

		defer file.Close()
		cid, err = g.ipfs.Add(file)
		if err != nil {
			return cid, err
		}
	}

	//Pin the file to prevent garbage collection
	err = g.ipfs.Pin(cid)
	if err != nil {
		return cid, err
	}

	return cid, nil
}

///Auth Transport for Infura

type authTransport struct {
	http.RoundTripper
	ProjectId     string
	ProjectSecret string
}

func (t authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(t.ProjectId, t.ProjectSecret)
	return t.RoundTripper.RoundTrip(r)
}

func getEnv(key string, _default string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return _default
}
