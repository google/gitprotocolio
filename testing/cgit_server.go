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

package testing

import (
	"io/ioutil"
	"net/http"
	"net/http/cgi"
)

// HTTPHandler returns an http.Handler that is backed by git-http-backend.
func HTTPHandler(gitBinary, gitDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h := &cgi.Handler{
			Path: gitBinary,
			Dir:  gitDir,
			Env: []string{
				"GIT_PROJECT_ROOT=" + gitDir,
				"GIT_HTTP_EXPORT_ALL=1",
			},
			Args: []string{
				"http-backend",
			},
			Stderr: ioutil.Discard,
		}
		if p := req.Header.Get("Git-Protocol"); p != "" {
			h.Env = append(h.Env, "GIT_PROTOCOL="+p)
		}
		if len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked" {
			// Not sure why this restriction is in place in the
			// library.
			req.TransferEncoding = nil
		}
		h.ServeHTTP(w, req)
	})
}
