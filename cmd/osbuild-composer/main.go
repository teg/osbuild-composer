package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/osbuild/osbuild-composer/internal/distro/fedora31"
	"github.com/osbuild/osbuild-composer/internal/distro/fedora32"
	"github.com/osbuild/osbuild-composer/internal/distro/rhel8"
	"github.com/osbuild/osbuild-composer/internal/jobqueue/fsjobqueue"

	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/distro"
	"github.com/osbuild/osbuild-composer/internal/rpmmd"
	"github.com/osbuild/osbuild-composer/internal/store"
	"github.com/osbuild/osbuild-composer/internal/weldr"
	"github.com/osbuild/osbuild-composer/internal/worker"

	"github.com/coreos/go-systemd/activation"
)

type connectionConfig struct {
	CACertFile     string
	ServerKeyFile  string
	ServerCertFile string
}

func createTLSConfig(c *connectionConfig) (*tls.Config, error) {
	caCertPEM, err := ioutil.ReadFile(c.CACertFile)
	if err != nil {
		return nil, err
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(caCertPEM)
	if !ok {
		panic("failed to parse root certificate")
	}

	cert, err := tls.LoadX509KeyPair(c.ServerCertFile, c.ServerKeyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    roots,
	}, nil
}

func main() {
	var verbose bool
	flag.BoolVar(&verbose, "v", false, "Print access log")
	flag.Parse()

	stateDir, ok := os.LookupEnv("STATE_DIRECTORY")
	if !ok {
		log.Fatal("STATE_DIRECTORY is not set. Is the service file missing StateDirectory=?")
	}

	listeners, err := activation.ListenersWithNames()
	if err != nil {
		log.Fatalf("Could not get listening sockets: " + err.Error())
	}

	if _, exists := listeners["osbuild-composer.socket"]; !exists {
		log.Fatalf("osbuild-composer.socket doesn't exist")
	}

	composerListeners := listeners["osbuild-composer.socket"]

	if len(composerListeners) != 2 && len(composerListeners) != 3 {
		log.Fatalf("Unexpected number of listening sockets (%d), expected 2 or 3", len(composerListeners))
	}

	weldrListener := composerListeners[0]
	jobListener := composerListeners[1]

	cacheDirectory, ok := os.LookupEnv("CACHE_DIRECTORY")
	if !ok {
		log.Fatal("CACHE_DIRECTORY is not set. Is the service file missing CacheDirectory=?")
	}

	rpm := rpmmd.NewRPMMD(path.Join(cacheDirectory, "rpmmd"), "/usr/libexec/osbuild-composer/dnf-json")

	distros, err := distro.NewRegistry(fedora31.New(), fedora32.New(), rhel8.New())
	if err != nil {
		log.Fatalf("Error loading distros: %v", err)
	}

	distribution, err := distros.FromHost()
	if err != nil {
		log.Fatalf("Could not determine distro from host: " + err.Error())
	}

	arch, err := distribution.GetArch(common.CurrentArch())
	if err != nil {
		log.Fatalf("Host distro does not support host architecture: " + err.Error())
	}

	repoMap, err := rpmmd.LoadRepositories([]string{"/etc/osbuild-composer", "/usr/share/osbuild-composer"}, distribution.Name())
	if err != nil {
		log.Fatalf("Could not load repositories for %s: %v", distribution.Name(), err)
	}

	var logger *log.Logger
	if verbose {
		logger = log.New(os.Stdout, "", 0)
	}

	store := store.New(&stateDir, arch, logger)

	queueDir := path.Join(stateDir, "jobs")
	err = os.Mkdir(queueDir, 0700)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("cannot create queue directory: %v", err)
	}

	jobs, err := fsjobqueue.New(queueDir, []string{"osbuild"})
	if err != nil {
		log.Fatalf("cannot create jobqueue: %v", err)
	}

	artifactsDir := path.Join(stateDir, "artifacts")
	err = os.Mkdir(artifactsDir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("cannot create artifacts directory: %v", err)
	}

	compatOutputDir := path.Join(stateDir, "outputs")

	workers := worker.NewServer(logger, jobs, artifactsDir)
	weldrAPI := weldr.New(rpm, arch, distribution, repoMap[common.CurrentArch()], logger, store, workers, compatOutputDir)

	go func() {
		err := workers.Serve(jobListener)
		common.PanicOnError(err)
	}()

	if remoteWorkerListeners, exists := listeners["osbuild-remote-worker.socket"]; exists {
		for _, listener := range remoteWorkerListeners {
			log.Printf("Starting remote listener\n")

			tlsConfig, err := createTLSConfig(&connectionConfig{
				CACertFile:     "/etc/osbuild-composer/ca-crt.pem",
				ServerKeyFile:  "/etc/osbuild-composer/composer-key.pem",
				ServerCertFile: "/etc/osbuild-composer/composer-crt.pem",
			})

			if err != nil {
				log.Fatalf("TLS configuration cannot be created: " + err.Error())
			}

			listener := tls.NewListener(listener, tlsConfig)
			go func() {
				err := workers.Serve(listener)
				common.PanicOnError(err)
			}()
		}
	}

	err = weldrAPI.Serve(weldrListener)
	common.PanicOnError(err)

}
