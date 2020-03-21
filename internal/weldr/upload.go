package weldr

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/distro"

	"github.com/google/uuid"
	"github.com/osbuild/osbuild-composer/internal/target"
)

type uploadResponse struct {
	Uuid         uuid.UUID              `json:"uuid"`
	Status       common.ImageBuildState `json:"status"`
	ProviderName string                 `json:"provider_name"`
	ImageName    string                 `json:"image_name"`
	CreationTime float64                `json:"creation_time"`
	Settings     uploadSettings         `json:"settings"`
}

type uploadSettings interface {
	isUploadSettings()
}

type awsUploadSettings struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
	Bucket          string `json:"bucket"`
	Key             string `json:"key"`
}

func (awsUploadSettings) isUploadSettings() {}

type azureUploadSettings struct {
	Account   string `json:"account"`
	AccessKey string `json:"accessKey"`
	Container string `json:"container"`
}

func (azureUploadSettings) isUploadSettings() {}

type uploadRequest struct {
	Provider  string         `json:"provider"`
	ImageName string         `json:"image_name"`
	Settings  uploadSettings `json:"settings"`
}

type rawUploadRequest struct {
	Provider  string          `json:"provider"`
	ImageName string          `json:"image_name"`
	Settings  json.RawMessage `json:"settings"`
}

func (u *uploadRequest) UnmarshalJSON(data []byte) error {
	var rawUploadRequest rawUploadRequest
	err := json.Unmarshal(data, &rawUploadRequest)
	if err != nil {
		return err
	}

	var settings uploadSettings
	switch rawUploadRequest.Provider {
	case "azure":
		settings = new(azureUploadSettings)
	case "aws":
		settings = new(awsUploadSettings)
	default:
		return errors.New("unexpected provider name")
	}
	err = json.Unmarshal(rawUploadRequest.Settings, settings)
	if err != nil {
		return err
	}

	u.Provider = rawUploadRequest.Provider
	u.ImageName = rawUploadRequest.ImageName
	u.Settings = settings

	return err
}

func targetsToUploadResponses(targets []*target.Target) []uploadResponse {
	var uploads []uploadResponse
	for _, t := range targets {
		upload := uploadResponse{
			Uuid:         t.Uuid,
			Status:       t.Status,
			ImageName:    t.ImageName,
			CreationTime: float64(t.Created.UnixNano()) / 1000000000,
		}

		switch options := t.Options.(type) {
		case *target.LocalTargetOptions:
			continue
		case *target.AWSTargetOptions:
			upload.ProviderName = "aws"
			upload.Settings = &awsUploadSettings{
				Region:          options.Region,
				AccessKeyID:     options.AccessKeyID,
				SecretAccessKey: options.SecretAccessKey,
				Bucket:          options.Bucket,
				Key:             options.Key,
			}
		case *target.AzureTargetOptions:
			upload.ProviderName = "azure"
			upload.Settings = &azureUploadSettings{
				Account:   options.Account,
				AccessKey: options.AccessKey,
				Container: options.Container,
			}
		}
		uploads = append(uploads, upload)
	}

	return uploads
}

func uploadRequestToTarget(u uploadRequest, d distro.Distro, imageType string) (*target.Target, error) {
	var t target.Target

	filename, _, err := d.FilenameFromType(imageType)
	if err != nil {
		return nil, err
	}

	t.Uuid = uuid.New()
	t.ImageName = u.ImageName
	t.Status = common.IBWaiting
	t.Created = time.Now()

	switch options := u.Settings.(type) {
	case *awsUploadSettings:
		t.Name = "org.osbuild.aws"
		t.Options = &target.AWSTargetOptions{
			Filename:        filename,
			Region:          options.Region,
			AccessKeyID:     options.AccessKeyID,
			SecretAccessKey: options.SecretAccessKey,
			Bucket:          options.Bucket,
			Key:             options.Key,
		}
	case *azureUploadSettings:
		t.Name = "org.osbuild.azure"
		t.Options = &target.AzureTargetOptions{
			Filename:  filename,
			Account:   options.Account,
			AccessKey: options.AccessKey,
			Container: options.Container,
		}
	}

	return &t, nil
}
