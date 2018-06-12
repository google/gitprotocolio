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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPush(t *testing.T) {
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	want, err := r.run("rev-parse", "master")
	if err != nil {
		t.Fatal(err)
	}

	for name, args := range protocolParams() {
		refreshRemote()
		if _, err := r.run(append(args, "push", httpProxyURL, "master:master")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got, err := remoteGitRepo.run("rev-parse", "master"); err != nil {
			t.Errorf("%s: %v", name, err)
		} else if got != want {
			t.Errorf("%s: want %s, got %s", name, want, got)
		}
	}
}

func TestPush_multipleRefs(t *testing.T) {
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("checkout", "-b", "another"); err != nil {
		t.Fatal(err)
	}

	for name, args := range protocolParams() {
		refreshRemote()
		if _, err := r.run(append(args, "push", httpProxyURL, "master:master", "another:another")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestPush_prereceiveReject(t *testing.T) {
	r := createLocalGitRepo()
	defer r.close()
	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	refreshRemote()
	if err := os.MkdirAll(filepath.Join(string(remoteGitRepo), "hooks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(string(remoteGitRepo), "hooks", "pre-receive"), []byte("#!/bin/sh\nfalse\n"), 0755); err != nil {
		t.Fatal(err)
	}

	for name, args := range protocolParams() {
		if _, err := r.run(append(args, "push", httpProxyURL, "master:master")...); err == nil {
			t.Errorf("%s: want error, got nothing", name)
			continue
		} else if cmderr, ok := err.(*commandError); !ok {
			t.Errorf("%s: want commandError, got %v", name, err)
			continue
		} else if !strings.Contains(cmderr.output, "pre-receive hook declined") {
			t.Errorf("%s: want a hook error, got %v", name, err)
			continue
		}
	}
}

func TestPush_nonFastForwardReject(t *testing.T) {
	r := createLocalGitRepo()
	defer r.close()
	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	refreshRemote()
	if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
		t.Fatalf("%v", err)
	}

	r = createLocalGitRepo()
	defer r.close()
	if _, err := r.run("commit", "--allow-empty", "--message=another"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("fetch", httpServerURL); err != nil {
		t.Fatal(err)
	}

	for name, args := range protocolParams() {
		if _, err := r.run(append(args, "push", httpProxyURL, "master:master")...); err == nil {
			t.Errorf("%s: want error, got nothing", name)
			continue
		} else if cmderr, ok := err.(*commandError); !ok {
			t.Errorf("%s: want commandError, got %v", name, err)
			continue
		} else if !strings.Contains(cmderr.output, "non-fast-forward") {
			t.Errorf("%s: want a non-fastforward error, got %v", name, err)
			continue
		}
	}
}

func TestPush_pushOption(t *testing.T) {
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}

	for name, args := range protocolParams() {
		refreshRemote()
		if _, err := r.run(append(args, "push", "-o", "testoption", httpProxyURL, "master:master")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestPush_multiplePushOptions(t *testing.T) {
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}

	for name, args := range protocolParams() {
		refreshRemote()
		if _, err := r.run(append(args, "push", "-o", "testoption1", "-o", "testoption2", httpProxyURL, "master:master")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestPush_shallowPush(t *testing.T) {
	for name, args := range protocolParams() {
		r := createLocalGitRepo()
		defer r.close()
		if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
			t.Fatal(err)
		}
		if _, err := r.run("commit", "--allow-empty", "--message=second"); err != nil {
			t.Fatal(err)
		}
		refreshRemote()
		if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
			t.Fatal(err)
		}

		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run("remote", "add", "origin", httpServerURL); err != nil {
			t.Fatal(err)
		}
		if _, err := r.run("pull", "--depth=1", "origin", "master"); err != nil {
			t.Fatal(err)
		}
		if _, err := r.run("commit", "--allow-empty", "--message=third"); err != nil {
			t.Fatal(err)
		}

		if _, err := r.run(append(args, "push", httpProxyURL, "master:master")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestPush_GPGSign(t *testing.T) {
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}

	for name, args := range protocolParams() {
		refreshRemote()
		if _, err := r.run(append(args, "push", "--signed", httpProxyURL, "master:master")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}
