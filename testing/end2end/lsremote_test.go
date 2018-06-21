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

func TestLsRemote_empty(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	for name, args := range protocolParams() {
		if _, err := r.run(append(args, "ls-remote", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestLsRemote_nonEmpty(t *testing.T) {
	refreshRemote()
	r := createLocalGitRepo()
	defer r.close()

	if _, err := r.run("commit", "--allow-empty", "--message=init"); err != nil {
		t.Fatal(err)
	}
	_, err := r.run("rev-parse", "master")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("push", httpServerURL, "master:master"); err != nil {
		t.Fatalf("%v", err)
	}

	r = createLocalGitRepo()
	for name, args := range protocolParams() {
		// TODO
		if _, err := r.run(append(args, "ls-remote", httpProxyURL)...); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
