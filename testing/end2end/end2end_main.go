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
package end2end

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/google/gitprotocolio/testing"
)

var (
	gitBinary     string
	remoteGitRepo gitRepo
	gnuPGHome     string

	httpServerURL string
	httpProxyURL  string
)

func init() {
	dir, err := ioutil.TempDir("", "gitprotocolio_remote")
	if err != nil {
		log.Fatal("cannot create remote dir", err)
	}
	remoteGitRepo = gitRepo(dir)

	gitBinary, err = exec.LookPath("git")
	if err != nil {
		log.Fatal("Cannot find the git binary: ", err)
	}

	{
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal(err)
		}
		httpBackend := &http.Server{
			Handler: testing.HTTPHandler(gitBinary, string(remoteGitRepo)),
		}
		go func() {
			log.Fatal(httpBackend.Serve(l))
		}()
		httpServerURL = fmt.Sprintf("http://%s/", l.Addr().String())
	}

	if os.Getenv("BYPASS_PROXY") == "1" {
		httpProxyURL = httpServerURL
	} else {
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal(err)
		}
		httpProxy := &http.Server{
			Handler: testing.HTTPProxyHandler(httpServerURL),
		}
		go func() {
			log.Fatal(httpProxy.Serve(l))
		}()
		httpProxyURL = fmt.Sprintf("http://%s/", l.Addr().String())
	}

	gnuPGHome, err = ioutil.TempDir("", "gitprotocolio_remote")
	if err != nil {
		log.Fatal("cannot create a GNUPGHOME: ", err)
	}

	gpgBinary, err := exec.LookPath("gpg")
	if err != nil {
		log.Fatal("cannot find the gpg binary: ", err)
	}

	cmd := exec.Command(gpgBinary, "--homedir", gnuPGHome, "--no-tty", "--batch", "--gen-key")
	cmd.Dir = gnuPGHome
	cmd.Stdin = bytes.NewBufferString(`
		%no-protection
		%transient-key
		Key-Type: RSA
		Key-Length: 2048
		Subkey-Type: RSA
		Subkey-Length: 2048
		Name-Real: local root
		Name-Email: local-root@example.com
		Expire-Date: 1d
	`)
	if bs, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("cannot create a GPG key: %v\n%s", err, string(bs))
	}

}

func refreshRemote() {
	remoteGitRepo.close()
	if err := os.Mkdir(string(remoteGitRepo), 0755); err != nil {
		log.Fatal(err)
	}
	remoteGitRepo.run("init", "--bare")
	remoteGitRepo.run("config", "http.receivepack", "1")
	remoteGitRepo.run("config", "uploadpack.allowfilter", "1")
	remoteGitRepo.run("config", "receive.advertisepushoptions", "1")
	remoteGitRepo.run("config", "receive.certNonceSeed", "testnonce")
}

func createLocalGitRepo() gitRepo {
	dir, err := ioutil.TempDir("", "gitprotocolio_local")
	if err != nil {
		log.Fatal(err)
	}
	r := gitRepo(dir)
	r.run("init")
	r.run("config", "user.email", "local-root@example.com")
	r.run("config", "user.name", "local root")
	return r
}

type gitRepo string

func (r gitRepo) run(arg ...string) (string, error) {
	cmd := exec.Command(gitBinary, arg...)
	cmd.Dir = string(r)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", gnuPGHome))
	bs, err := cmd.CombinedOutput()
	if err != nil {
		return "", &commandError{err, cmd.Args, strings.TrimRight(string(bs), "\n")}
	}
	return string(bs), nil
}

func (r gitRepo) close() error {
	return os.RemoveAll(string(r))
}

type commandError struct {
	err    error
	args   []string
	output string
}

func (c *commandError) Error() string {
	ss := []string{
		"cannot execute a git command",
		fmt.Sprintf("Error: %v", c.err),
		fmt.Sprintf("Args: %#v", c.args),
	}
	for _, s := range strings.Split(c.output, "\n") {
		ss = append(ss, "Output: "+s)
	}
	return strings.Join(ss, "\n")
}
