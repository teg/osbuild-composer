package worker

import (
	"github.com/google/uuid"

	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/distro"
	"github.com/osbuild/osbuild-composer/internal/target"
)

//
// JSON-serializable types for the jobqueue
//

type buildJob struct {
	Manifest distro.Manifest  `json:"manifest"`
	Targets  []*target.Target `json:"targets,omitempty"`
}

type buildJobResult struct {
	OSBuildOutput *common.ComposeResult `json:"osbuild_output,omitempty"`
}

type registrationJob struct {
	Targets []*target.Target `json:"targets,omitempty"`
}

type registrationJobResult struct {
	RegistrationOutput *common.ComposeResult `json:"registration_output,omitempty"`
}

//
// JSON-serializable types for the HTTP API
//

type statusResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Message string `json:"message"`
}

type addJobRequest struct {
	JobType string `json:"job_type"`
}

type addBuildJobResponse struct {
	ID       uuid.UUID        `json:"id"`
	Manifest distro.Manifest  `json:"manifest"`
	Targets  []*target.Target `json:"targets,omitempty"`
}

type addRegistrationJobResponse struct {
	ID           uuid.UUID              `json:"id"`
	BuildResults []common.ComposeResult `json:"build_results"`
	Targets      []*target.Target       `json:"targets,omitempty"`
}

type jobResponse struct {
	ID       uuid.UUID `json:"id"`
	Canceled bool      `json:"canceled"`
}

type updateJobRequest struct {
	Status common.ImageBuildState `json:"status"`
	Result *common.ComposeResult  `json:"result"`
}

type updateJobResponse struct {
}
