// Testproxy is a Git protocol HTTP transport proxy.
//
// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/google/gitprotocolio/testing"
)

var (
	port        = flag.Int("port", 0, "the proxy port number")
	delegateURL = flag.String("delegate_url", "", "the Git repository URL the server delegates to")
)

func main() {
	flag.Parse()

	if *port == 0 {
		log.Fatal("--port is unspecified")
	}
	if *delegateURL == "" {
		log.Fatal("--delegate_url is unspecified")
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), testing.HTTPProxyHandler(*delegateURL)))
}
