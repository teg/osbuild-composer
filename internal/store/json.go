package store

import (
	"errors"
	"sort"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/google/uuid"
	"github.com/osbuild/osbuild-composer/internal/blueprint"
	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/distro"
	"github.com/osbuild/osbuild-composer/internal/osbuild"
	"github.com/osbuild/osbuild-composer/internal/target"
)

type storeV0 struct {
	Blueprints blueprintsV0 `json:"blueprints"`
	Workspace  workspaceV0  `json:"workspace"`
	Composes   composesV0   `json:"composes"`
	Sources    sourcesV0    `json:"sources"`
	Changes    changesV0    `json:"changes"`
	Commits    commitsV0    `json:"commits"`
}

type blueprintsV0 map[string]blueprint.Blueprint
type workspaceV0 map[string]blueprint.Blueprint

// A Compose represent the task of building a set of images from a single blueprint.
// It contains all the information necessary to generate the inputs for the job, as
// well as the job's state.
type composeV0 struct {
	Blueprint   *blueprint.Blueprint `json:"blueprint"`
	ImageBuilds []imageBuildV0       `json:"image_builds"`
}

type composesV0 map[uuid.UUID]composeV0

// ImageBuild represents a single image build inside a compose
type imageBuildV0 struct {
	ID          int               `json:"id"`
	ImageType   string            `json:"image_type"`
	Manifest    *osbuild.Manifest `json:"manifest"`
	Targets     []*target.Target  `json:"targets"`
	JobCreated  time.Time         `json:"job_created"`
	JobStarted  time.Time         `json:"job_started"`
	JobFinished time.Time         `json:"job_finished"`
	Size        uint64            `json:"size"`
	JobID       uuid.UUID         `json:"jobid,omitempty"`

	// Kept for backwards compatibility. Image builds which were done
	// before the move to the job queue use this to store whether they
	// finished successfully.
	QueueStatus common.ImageBuildState `json:"queue_status,omitempty"`
}

type sourceV0 struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	CheckGPG bool   `json:"check_gpg"`
	CheckSSL bool   `json:"check_ssl"`
	System   bool   `json:"system"`
}

type sourcesV0 map[string]sourceV0

type changeV0 struct {
	Commit    string `json:"commit"`
	Message   string `json:"message"`
	Revision  *int   `json:"revision"`
	Timestamp string `json:"timestamp"`
}

type changesV0 map[string]map[string]changeV0

type commitsV0 map[string][]string

func newBlueprintsFromV0(blueprintsStruct blueprintsV0) map[string]blueprint.Blueprint {
	blueprints := make(map[string]blueprint.Blueprint)
	for name, blueprint := range blueprintsStruct {
		blueprints[name] = blueprint.DeepCopy()
	}
	return blueprints
}

func newWorkspaceFromV0(workspaceStruct workspaceV0) map[string]blueprint.Blueprint {
	workspace := make(map[string]blueprint.Blueprint)
	for name, blueprint := range workspaceStruct {
		workspace[name] = blueprint.DeepCopy()
	}
	return workspace
}

func newComposesFromV0(composesStruct composesV0, arch distro.Arch) map[uuid.UUID]Compose {
	composes := make(map[uuid.UUID]Compose)

	for composeID, composeStruct := range composesStruct {
		c, err := newComposeFromV0(composeStruct, arch)
		if err != nil {
			// Ignore invalid composes.
			continue
		}
		composes[composeID] = c
	}

	return composes
}

func newComposeFromV0(composeStruct composeV0, arch distro.Arch) (Compose, error) {
	c := Compose{
		Blueprint: composeStruct.Blueprint,
	}
	for _, imgBuild := range composeStruct.ImageBuilds {
		imgType := imageTypeFromCompatString(imgBuild.ImageType, arch)
		if imgType == nil {
			// Invalid type strings in serialization format, this may happen
			// on upgrades.
			return Compose{}, errors.New("Invalid Image Type string.")
		}
		ib := ImageBuild{
			ID:          imgBuild.ID,
			ImageType:   imgType,
			Manifest:    imgBuild.Manifest,
			Targets:     imgBuild.Targets,
			JobCreated:  imgBuild.JobCreated,
			JobStarted:  imgBuild.JobStarted,
			JobFinished: imgBuild.JobFinished,
			Size:        imgBuild.Size,
			JobID:       imgBuild.JobID,
			QueueStatus: imgBuild.QueueStatus,
		}
		c.ImageBuild = ib
		return c, nil
	}
	return Compose{}, errors.New("No valid image build found in compose.")
}

func newSourceConfigsFromV0(sourcesStruct sourcesV0) map[string]SourceConfig {
	sources := make(map[string]SourceConfig)

	for name, source := range sourcesStruct {
		sources[name] = SourceConfig(source)
	}

	return sources
}

func newChangesFromV0(changesStruct changesV0) map[string]map[string]blueprint.Change {
	changes := make(map[string]map[string]blueprint.Change)

	for name, commitsStruct := range changesStruct {
		commits := make(map[string]blueprint.Change)
		for commitID, change := range commitsStruct {
			commits[commitID] = blueprint.Change{
				Commit:    change.Commit,
				Message:   change.Message,
				Revision:  change.Revision,
				Timestamp: change.Timestamp,
			}
		}
		changes[name] = commits
	}

	return changes
}

func newCommitsFromV0(commitsStruct commitsV0) map[string][]string {
	commits := make(map[string][]string)
	for name, changes := range commitsStruct {
		commits[name] = changes
	}
	return commits
}

func newStoreFromV0(storeStruct storeV0, arch distro.Arch) *Store {
	store := Store{
		blueprints:        newBlueprintsFromV0(storeStruct.Blueprints),
		workspace:         newWorkspaceFromV0(storeStruct.Workspace),
		composes:          newComposesFromV0(storeStruct.Composes, arch),
		sources:           newSourceConfigsFromV0(storeStruct.Sources),
		blueprintsChanges: newChangesFromV0(storeStruct.Changes),
		blueprintsCommits: newCommitsFromV0(storeStruct.Commits),
	}

	// Backwards compatibility: fail all builds that are queued or
	// running. Jobs status is now handled outside of the store
	// (and the compose). The fields are kept so that previously
	// succeeded builds still show up correctly.
	for composeID, compose := range store.composes {
		switch compose.ImageBuild.QueueStatus {
		case common.IBRunning, common.IBWaiting:
			compose.ImageBuild.QueueStatus = common.IBFailed
			store.composes[composeID] = compose

		}
	}

	// Populate BlueprintsCommits for existing blueprints without commit history
	// BlueprintsCommits tracks the order of the commits in BlueprintsChanges,
	// but may not be in-sync with BlueprintsChanges because it was added later.
	// This will sort the existing commits by timestamp and version to update
	// the store. BUT since the timestamp resolution is only 1s it is possible
	// that the order may be slightly wrong.
	for name := range store.blueprintsChanges {
		if len(store.blueprintsChanges[name]) != len(store.blueprintsCommits[name]) {
			changes := make([]blueprint.Change, 0, len(store.blueprintsChanges[name]))

			for commit := range store.blueprintsChanges[name] {
				changes = append(changes, store.blueprintsChanges[name][commit])
			}

			// Sort the changes by Timestamp then version, ascending
			sort.Slice(changes, func(i, j int) bool {
				if changes[i].Timestamp == changes[j].Timestamp {
					vI, err := semver.NewVersion(changes[i].Blueprint.Version)
					if err != nil {
						vI = semver.New("0.0.0")
					}
					vJ, err := semver.NewVersion(changes[j].Blueprint.Version)
					if err != nil {
						vJ = semver.New("0.0.0")
					}

					return vI.LessThan(*vJ)
				}
				return changes[i].Timestamp < changes[j].Timestamp
			})

			commits := make([]string, 0, len(changes))
			for _, c := range changes {
				commits = append(commits, c.Commit)
			}

			store.blueprintsCommits[name] = commits
		}
	}

	return &store
}

func newComposesV0(composes map[uuid.UUID]Compose) composesV0 {
	composesStruct := make(composesV0)
	for composeID, compose := range composes {
		c := composeV0{
			Blueprint: compose.Blueprint,
		}
		imgType := imageTypeToCompatString(compose.ImageBuild.ImageType)
		if imgType == "" {
			panic("invalid image type: " + compose.ImageBuild.ImageType.Name())
		}
		ib := imageBuildV0{
			ID:          compose.ImageBuild.ID,
			ImageType:   imgType,
			Manifest:    compose.ImageBuild.Manifest,
			Targets:     compose.ImageBuild.Targets,
			JobCreated:  compose.ImageBuild.JobCreated,
			JobStarted:  compose.ImageBuild.JobStarted,
			JobFinished: compose.ImageBuild.JobFinished,
			Size:        compose.ImageBuild.Size,
			JobID:       compose.ImageBuild.JobID,
			QueueStatus: compose.ImageBuild.QueueStatus,
		}
		c.ImageBuilds = append(c.ImageBuilds, ib)
		composesStruct[composeID] = c
	}
	return composesStruct
}

func newSourcesV0(sources map[string]SourceConfig) sourcesV0 {
	sourcesStruct := make(sourcesV0)
	for name, source := range sources {
		sourcesStruct[name] = sourceV0(source)
	}
	return sourcesStruct
}

func newChangesV0(changes map[string]map[string]blueprint.Change) changesV0 {
	changesStruct := make(changesV0)
	for name, commits := range changes {
		commitsStruct := make(map[string]changeV0)
		for commitID, change := range commits {
			commitsStruct[commitID] = changeV0{
				Commit:    change.Commit,
				Message:   change.Message,
				Revision:  change.Revision,
				Timestamp: change.Timestamp,
			}
		}
		changesStruct[name] = commitsStruct
	}
	return changesStruct
}

func newCommitsV0(commits map[string][]string) commitsV0 {
	commitsStruct := make(commitsV0)
	for name, changes := range commits {
		commitsStruct[name] = changes
	}
	return commitsStruct
}

func (store *Store) toStoreV0() *storeV0 {
	return &storeV0{
		Blueprints: store.blueprints,
		Workspace:  store.workspace,
		Composes:   newComposesV0(store.composes),
		Sources:    newSourcesV0(store.sources),
		Changes:    newChangesV0(store.blueprintsChanges),
		Commits:    newCommitsV0(store.blueprintsCommits),
	}
}

var imageTypeCompatMapping = map[string]string{
	"vhd":              "Azure",
	"ami":              "AWS",
	"liveiso":          "LiveISO",
	"openstack":        "OpenStack",
	"qcow2":            "qcow2",
	"vmdk":             "VMWare",
	"ext4-filesystem":  "Raw-filesystem",
	"partitioned-disk": "Partitioned-disk",
	"tar":              "Tar",
	"test_type":        "test_type",
}

func imageTypeToCompatString(imgType distro.ImageType) string {
	return imageTypeCompatMapping[imgType.Name()]
}

func imageTypeFromCompatString(input string, arch distro.Arch) distro.ImageType {
	for k, v := range imageTypeCompatMapping {
		if v == input {
			imgType, err := arch.GetImageType(k)
			if err != nil {
				return nil
			}
			return imgType
		}
	}
	return nil
}
