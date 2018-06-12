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
	"testing"
)

func TestFetch(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	want, err := r.run("rev-parse", "master")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
		t.Fatalf("%v", err)
	}

	for name, args := range protocolParams() {
		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run(append(args, "fetch", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got, err := r.run("rev-parse", "FETCH_HEAD"); err != nil {
			t.Errorf("%s: %v", name, err)
		} else if got != want {
			t.Errorf("%s: want %s, got %s", name, want, got)
		}
	}
}

func TestFetch_multiWants(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("checkout", "--orphan", "testbranch"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("commit", "--allow-empty", "--message=another"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "master:master", "testbranch:testbranch"); err != nil {
		t.Fatalf("%v", err)
	}

	for name, args := range protocolParams() {
		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run(append(args, "remote", "add", "origin", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if _, err := r.run(append(args, "fetch", "origin")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestFetch_shallowDepth(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	want, err := r.run("rev-parse", "master")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
		t.Fatalf("%v", err)
	}

	for name, args := range protocolParams() {
		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run(append(args, "fetch", "--depth=1", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got, err := r.run("rev-parse", "FETCH_HEAD"); err != nil {
			t.Errorf("%s: %v", name, err)
		} else if got != want {
			t.Errorf("%s: want %s, got %s", name, want, got)
		}
	}
}

func TestFetch_deepen(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
		t.Fatalf("%v", err)
	}

	for name, args := range protocolParams() {
		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run(append(args, "remote", "add", "origin", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if _, err := r.run(append(args, "fetch", "--depth=1", "origin")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if _, err := r.run(append(args, "fetch", "--deepen=1", "origin")...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestFetch_shallowSince(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
		t.Fatalf("%v", err)
	}

	for name, args := range protocolParams() {
		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run(append(args, "fetch", "--shallow-since=2000-01-01", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestFetch_shallowNot(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("tag", "-a", "should_be_excluded", "-m", "excluded"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("commit", "--allow-empty", "--message=second"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "--follow-tags"); err != nil {
		t.Fatalf("%v", err)
	}

	for name, args := range protocolParams() {
		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run(append(args, "fetch", "--shallow-exclude=should_be_excluded", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}

func TestFetch_filter(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
		t.Fatalf("%v", err)
	}

	for name, args := range protocolParams() {
		r = createLocalGitRepo()
		defer r.close()
		if _, err := r.run("config", "extensions.partialClone", httpProxyURL); err != nil {
			t.Fatalf("%v", err)
		}
		if _, err := r.run(append(args, "fetch", "--filter=blob:none", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
	}
}
