package worker

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/google/uuid"

	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/distro"
	"github.com/osbuild/osbuild-composer/internal/target"
)

type Client struct {
	client   *http.Client
	scheme   string
	hostname string
}

type BuildJob struct {
	ID       uuid.UUID
	Manifest distro.Manifest
	Targets  []*target.Target
}

type RegistrationJob struct {
	ID           uuid.UUID
	BuildResults []common.ComposeResult
	Targets      []*target.Target
}

func NewClient(address string, conf *tls.Config) *Client {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: conf,
		},
	}

	var scheme string
	// Use https if the TLS configuration is present, otherwise use http.
	if conf != nil {
		scheme = "https"
	} else {
		scheme = "http"
	}

	return &Client{client, scheme, address}
}

func NewClientUnix(path string) *Client {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(context context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", path)
			},
		},
	}

	return &Client{client, "http", "localhost"}
}

func (c *Client) AddBuildJob() (*BuildJob, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(addJobRequest{JobType: "osbuild"})
	if err != nil {
		panic(err)
	}
	response, err := c.client.Post(c.createURL("/job-queue/v1/jobs"), "application/json", &b)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		var er errorResponse
		_ = json.NewDecoder(response.Body).Decode(&er)
		return nil, fmt.Errorf("couldn't create job, got %d: %s", response.StatusCode, er.Message)
	}

	var jr addBuildJobResponse
	err = json.NewDecoder(response.Body).Decode(&jr)
	if err != nil {
		return nil, err
	}

	return &BuildJob{
		jr.ID,
		jr.Manifest,
		jr.Targets,
	}, nil
}

func (c *Client) AddRegistrationJob() (*RegistrationJob, error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(addJobRequest{JobType: "registration"})
	if err != nil {
		panic(err)
	}
	response, err := c.client.Post(c.createURL("/job-queue/v1/jobs"), "application/json", &b)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		var er errorResponse
		_ = json.NewDecoder(response.Body).Decode(&er)
		return nil, fmt.Errorf("couldn't create job, got %d: %s", response.StatusCode, er.Message)
	}

	var jr addRegistrationJobResponse
	err = json.NewDecoder(response.Body).Decode(&jr)
	if err != nil {
		return nil, err
	}

	return &RegistrationJob{
		jr.ID,
		jr.BuildResults,
		jr.Targets,
	}, nil
}

func (c *Client) JobCanceled(jobID uuid.UUID) bool {
	response, err := c.client.Get(c.createURL("/job-queue/v1/jobs/" + jobID.String()))
	if err != nil {
		return true
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return true
	}

	var jr jobResponse
	err = json.NewDecoder(response.Body).Decode(&jr)
	if err != nil {
		return true
	}

	return jr.Canceled
}

func (c *Client) UpdateJob(job *BuildJob, status common.ImageBuildState, result *common.ComposeResult) error {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(&updateJobRequest{status, result})
	if err != nil {
		panic(err)
	}
	urlPath := fmt.Sprintf("/job-queue/v1/jobs/%s", job.ID)
	url := c.createURL(urlPath)
	req, err := http.NewRequest("PATCH", url, &b)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New("error setting job status")
	}

	return nil
}

func (c *Client) UploadImage(job uuid.UUID, name string, reader io.Reader) error {
	url := c.createURL(fmt.Sprintf("/job-queue/v1/jobs/%s/artifacts/%s", job, name))
	_, err := c.client.Post(url, "application/octet-stream", reader)

	return err
}

func (c *Client) createURL(path string) string {
	return c.scheme + "://" + c.hostname + path
}
