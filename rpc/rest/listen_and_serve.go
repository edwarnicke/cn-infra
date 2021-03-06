// Copyright (c) 2017 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// ListenAndServe is a function that uses <config> & <handler> to handle
// HTTP Requests.
// It return an instance of io.Closer to close the HTTP Server during cleanup.
type ListenAndServe func(config Config, handler http.Handler) (
	httpServer io.Closer, err error)

// FromExistingServer is used mainly for testing purposes
//
// Example:
//
//    httpmux.FromExistingServer(mock.SetHandler)
//	  mock.NewRequest("GET", "/v1/a", nil)
//
func FromExistingServer(listenAndServe ListenAndServe) *Plugin {
	return &Plugin{listenAndServe: listenAndServe}
}

// ListenAndServeHTTP starts a http server.
func ListenAndServeHTTP(config Config, handler http.Handler) (httpServer io.Closer, err error) {

	tlsCfg := &tls.Config{}

	if len(config.ClientCerts) > 0 {
		// require client certificate
		caCertPool := x509.NewCertPool()

		for _, c := range config.ClientCerts {
			caCert, err := ioutil.ReadFile(c)
			if err != nil {
				return nil, err
			}
			caCertPool.AppendCertsFromPEM(caCert)
		}

		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		tlsCfg.ClientCAs = caCertPool
	}

	server := &http.Server{
		Addr:              config.Endpoint,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
		TLSConfig:         tlsCfg,
	}
	server.Handler = handler

	var errCh chan error
	go func() {
		var err error
		if config.UseHTTPS() {
			// if server certificate and key is configured use HTTPS
			err = server.ListenAndServeTLS(config.ServerCertfile, config.ServerKeyfile)
		} else {
			err = server.ListenAndServe()
		}

		errCh <- err

	}()

	select {
	case err := <-errCh:
		return nil, err
		// Wait 100ms to create a new stream, so it doesn't bring too much
		// overhead when retry.
	case <-time.After(100 * time.Millisecond):
		//everything is probably fine
		return server, nil
	}
}
