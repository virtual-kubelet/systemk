// Copyright © 2017 The virtual-kubelet authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Edited by the systemk authors in 2021.

package cmd

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/virtual-kubelet/systemk/internal/provider"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
)

// AcceptedCiphers is the list of accepted TLS ciphers, with known weak ciphers elided
// Note this list should be a moving target.
var AcceptedCiphers = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,

	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
}

func loadTLSConfig(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, errors.Wrap(err, "error loading tls certs")
	}

	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites:             AcceptedCiphers,
	}, nil
}

// setupKubeletServer configures and brings up the kubelet API server.
func setupKubeletServer(ctx context.Context, config *provider.Opts, p provider.Provider, getPodsFromKubernetes nodeapi.PodListerFunc) (_ func(), retErr error) {
	var closers []io.Closer
	cancel := func() {
		for _, c := range closers {
			c.Close()
		}
	}
	defer func() {
		if retErr != nil {
			cancel()
		}
	}()

	// Ensure valid TLS setup.
	if config.ServerCertPath == "" || config.ServerKeyPath == "" {
		log.
			WithField("cert", config.ServerCertPath).
			WithField("key", config.ServerKeyPath).
			Error("TLS certificates are required to serve the kubelet API")
	} else {
		tlsCfg, err := loadTLSConfig(config.ServerCertPath, config.ServerKeyPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load TLS required for serving the kubelet API")
		}
		l, err := tls.Listen("tcp", config.ListenAddress, tlsCfg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup the kubelet API server")
		}

		// Setup path routing.
		r := mux.NewRouter()

		// This matches the behaviour in the reference kubelet
		r.StrictSlash(true)

		// Setup routes.
		r.HandleFunc("/pods", nodeapi.HandleRunningPods(getPodsFromKubernetes)).Methods("GET")
		r.HandleFunc("/containerLogs/{namespace}/{pod}/{container}", p.GetContainerLogsHandler).Methods("GET")
		r.HandleFunc(
			"/exec/{namespace}/{pod}/{container}",
			nodeapi.HandleContainerExec(
				p.RunInContainer,
				nodeapi.WithExecStreamCreationTimeout(config.StreamCreationTimeout),
				nodeapi.WithExecStreamIdleTimeout(config.StreamIdleTimeout),
			),
		).Methods("POST", "GET")

		// TODO(pires) uncomment this when VK imports k8s.io/kubelet v0.20+
		//if p.GetStatsSummary != nil {
		//	f := nodeapi.HandlePodStatsSummary(p.GetStatsSummary)
		//	r.HandleFunc("/stats/summary", f).Methods("GET")
		//	r.HandleFunc("/stats/summary/", f).Methods("GET")
		//}

		r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		})

		// Start the server.
		s := &http.Server{
			Handler:   r,
			TLSConfig: tlsCfg,
		}
		go serveHTTP(ctx, s, l)
		closers = append(closers, s)
	}

	// TODO(pires) metrics are disabled until VK supports k8s.io/kubelet v0.20+
	// This is so that we don't import k8s.io/kubernetes.
	//
	// We're keeping the code commented so to not forget and to make it easier
	// to enable later.
	//if cfg.MetricsAddr == "" {
	//	log.G(ctx).Info("pod metrics server not setup due to empty metrics address")
	//} else {
	//	l, err := net.Listen("tcp", cfg.MetricsAddr)
	//	if err != nil {
	//		return nil, errors.Wrap(err, "could not setup listener for pod metrics http server")
	//	}
	//
	//	mux := http.NewServeMux()
	//
	//	var summaryHandlerFunc api.PodStatsSummaryHandlerFunc
	//	if mp, ok := p.(provider.PodMetricsProvider); ok {
	//		summaryHandlerFunc = mp.GetStatsSummary
	//	}
	//	podMetricsRoutes := api.PodMetricsConfig{
	//		GetStatsSummary: summaryHandlerFunc,
	//	}
	//	api.AttachPodMetricsRoutes(podMetricsRoutes, mux)
	//	s := &http.Server{
	//		Handler: mux,
	//	}
	//	go serveHTTP(ctx, s, l, "pod metrics")
	//	closers = append(closers, s)
	//}

	return cancel, nil
}

func serveHTTP(ctx context.Context, s *http.Server, l net.Listener) {
	if err := s.Serve(l); err != nil {
		select {
		case <-ctx.Done():
		default:
			log.WithError(err).Error("failed to setup the kubelet API server")
		}
	}
	l.Close()
}
