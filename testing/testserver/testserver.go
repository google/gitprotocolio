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

// testserver runs cgit servers.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/google/gitprotocolio/testing"
)

var (
	httpPort = flag.Int("http_port", 0, "the HTTP port number")
)

func main() {
	flag.Parse()

	gitDir, err := ioutil.TempDir("", "gitprotocolio_testserver")
	if err != nil {
		log.Fatal("cannot create remote dir", err)
	}
	defer func() {
		if err := os.RemoveAll(gitDir); err != nil {
			log.Fatal(err)
		}
	}()

	gitBinary, err := exec.LookPath("git")
	if err != nil {
		log.Fatal("Cannot find the git binary: ", err)
	}

	runGit(gitBinary, gitDir, "init", "--bare")
	runGit(gitBinary, gitDir, "config", "http.receivepack", "1")
	runGit(gitBinary, gitDir, "config", "uploadpack.allowfilter", "1")
	runGit(gitBinary, gitDir, "config", "receive.advertisepushoptions", "1")
	runGit(gitBinary, gitDir, "config", "receive.certNonceSeed", "testnonce")

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", *httpPort))
	if err != nil {
		log.Fatal(err)
	}
	httpBackend := &http.Server{
		Handler: testing.HTTPHandler(gitBinary, gitDir),
	}
	log.Println(gitDir)
	log.Println(fmt.Sprintf("http://%s/", l.Addr().String()))
	log.Fatal(httpBackend.Serve(l))
}

func runGit(gitBinary, gitDir string, arg ...string) {
	cmd := exec.Command(gitBinary, arg...)
	cmd.Dir = gitDir
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
