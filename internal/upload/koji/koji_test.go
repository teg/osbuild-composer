//+build koji_test

package koji_test

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/osbuild-composer/internal/upload/koji"
)

func TestKojiImport(t *testing.T) {
	// define constants
	server := "https://localhost/kojihub"
	filename := "image.qcow2"
	filesize := 1024
	shareDir := "/tmp/osbuild-composer-koji-test"
	// you cannot create two build with a same name, let's create a random one each time
	buildName := "osbuild-image-" + uuid.Must(uuid.NewRandom()).String()
	// koji needs to specify a directory to which the upload should happen, let's reuse the build name
	uploadDirectory := buildName

	// base our transport on the default one
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// use the self-signed certificate generated by run-koji-container
	certPool := x509.NewCertPool()
	cert, err := ioutil.ReadFile(shareDir + "/ca-crt.pem")
	require.NoError(t, err)

	ok := certPool.AppendCertsFromPEM(cert)
	require.True(t, ok)

	transport.TLSClientConfig = &tls.Config{
		RootCAs: certPool,
	}

	// login
	credentials := &koji.GSSAPICredentials{
		Principal: "osbuild-krb@LOCAL",
		KeyTab:    shareDir + "/client.keytab",
	}
	k, err := koji.NewFromGSSAPI(server, credentials, transport)
	require.NoError(t, err)

	defer func() {
		err := k.Logout()
		if err != nil {
			require.NoError(t, err)
		}
	}()

	// Create a random file
	f, err := ioutil.TempFile("", "osbuild-koji-test-*.qcow2")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, f.Close())
		assert.NoError(t, os.Remove(f.Name()))
	}()

	_, err = io.CopyN(f, rand.Reader, int64(filesize))
	require.NoError(t, err)
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)

	// Upload the file
	hash, _, err := k.Upload(f, uploadDirectory, filename)
	require.NoError(t, err)

	// Import the build
	build := koji.ImageBuild{
		Name:      buildName,
		Version:   "1",
		Release:   "1",
		StartTime: time.Now().Unix(),
		EndTime:   time.Now().Unix(),
	}
	buildRoots := []koji.BuildRoot{
		{
			ID: 1,
			Host: koji.Host{
				Os:   "RHEL8",
				Arch: "noarch",
			},
			ContentGenerator: koji.ContentGenerator{
				Name:    "osbuild",
				Version: "1",
			},
			Container: koji.Container{
				Type: "nspawn",
				Arch: "noarch",
			},
			Tools: []koji.Tool{},
			RPMs:  []koji.RPM{},
		},
	}
	output := []koji.Image{
		{
			BuildRootID:  1,
			Filename:     filename,
			FileSize:     uint64(filesize),
			Arch:         "noarch",
			ChecksumType: "md5",
			MD5:          hash,
			Type:         "image",
			RPMs:         []koji.RPM{},
			Extra: koji.ImageExtra{
				Info: koji.ImageExtraInfo{
					Arch: "noarch",
				},
			},
		},
	}

	initResult, err := k.CGInitBuild(build.Name, build.Version, build.Release)
	require.NoError(t, err)

	build.BuildID = uint64(initResult.BuildID)

	importResult, err := k.CGImport(build, buildRoots, output, uploadDirectory, initResult.Token)
	require.NoError(t, err)

	// check if the build is really there:
	cmd := exec.Command(
		"koji",
		"--server", server,
		"-c", "../../../.github/koji.conf",
		"--keytab", credentials.KeyTab,
		"--principal", credentials.Principal,
		"list-builds",
		"--buildid", strconv.Itoa(importResult.BuildID),
	)

	// sample output:
	// Build                                                    Built by          State
	// -------------------------------------------------------  ----------------  ----------------
	// osbuild-image-92882b90-4bd9-4422-8b8a-40863f94535a-1-1   osbuild           COMPLETE
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err)

	// let's check for COMPLETE, koji will exit with non-zero status code if the build doesn't exist
	assert.Contains(t, string(out), "COMPLETE")
}
