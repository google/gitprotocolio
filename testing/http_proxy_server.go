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
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"sync"

	"github.com/google/gitprotocolio"
)

// HTTPProxyHandler returns an http.handler that delegates requests to the
// provided URL.
func HTTPProxyHandler(delegateURL string) http.Handler {
	s := &httpProxyServer{delegateURL}
	mux := http.NewServeMux()
	mux.HandleFunc("/info/refs", s.infoRefsHandler)
	mux.HandleFunc("/git-upload-pack", s.uploadPackHandler)
	mux.HandleFunc("/git-receive-pack", s.receivePackHandler)
	return mux
}

type httpProxyServer struct {
	delegateURL string
}

func (s *httpProxyServer) infoRefsHandler(w http.ResponseWriter, r *http.Request) {
	u, err := httpURLForLsRemote(s.delegateURL, r.URL.Query().Get("service"))
	if err != nil {
		http.Error(w, "cannot construct the /info/refs URL", http.StatusInternalServerError)
		log.Printf("cannot construct the /info/refs URL: %#v", err)
		return
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		http.Error(w, "cannot construct the request object", http.StatusInternalServerError)
		return
	}
	req.Header.Add("Accept", "*/*")
	if proto := r.Header.Get("Git-Protocol"); proto == "version=2" || proto == "version=1" {
		req.Header.Add("Git-Protocol", proto)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "cannot send a request to the delegate", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	w.Header().Add("Content-Type", fmt.Sprintf("application/x-%s-advertisement", r.URL.Query().Get("service")))
	infoRefsResp := gitprotocolio.NewInfoRefsResponse(resp.Body)
	for infoRefsResp.Scan() {
		if err := writePacket(w, infoRefsResp.Chunk()); err != nil {
			writePacket(w, gitprotocolio.ErrorPacket("cannot write a packet"))
			return
		}
	}

	if err := infoRefsResp.Err(); err != nil {
		if ep, ok := err.(gitprotocolio.ErrorPacket); ok {
			writePacket(w, ep)
		} else {
			writePacket(w, gitprotocolio.ErrorPacket("internal error"))
			log.Printf("Parsing error: %#v, parser: %#v", err, infoRefsResp)
		}
		return
	}
}

func (s *httpProxyServer) uploadPackHandler(w http.ResponseWriter, r *http.Request) {
	u, err := httpURLForUploadPack(s.delegateURL)
	if err != nil {
		http.Error(w, "cannot construct the /git-upload-pack URL", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Content-Encoding") == "gzip" {
		var err error
		if r.Body, err = gzip.NewReader(r.Body); err != nil {
			http.Error(w, "cannot ungzip", http.StatusBadRequest)
			return
		}
	}

	if r.Header.Get("Git-Protocol") == "version=2" {
		serveProtocolV2(u, w, r)
		return
	}
	uploadPackV1Handler(u, w, r)
}

func uploadPackV1Handler(delegateURL string, w http.ResponseWriter, r *http.Request) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		v1Req := gitprotocolio.NewProtocolV1UploadPackRequest(r.Body)

		for v1Req.Scan() {
			if err := writePacket(pw, v1Req.Chunk()); err != nil {
				writePacket(pw, gitprotocolio.ErrorPacket("cannot write a packet"))
				return
			}
		}

		if err := v1Req.Err(); err != nil {
			if ep, ok := err.(gitprotocolio.ErrorPacket); ok {
				writePacket(pw, ep)
			} else {
				writePacket(pw, gitprotocolio.ErrorPacket("internal error"))
				log.Printf("Parsing error: %#v, parser: %#v", err, v1Req)
			}
			return
		}
	}()

	req, err := http.NewRequest("POST", delegateURL, pr)
	if err != nil {
		http.Error(w, "cannot construct the request object", http.StatusInternalServerError)
		return
	}
	req.Header.Add("Content-Type", "application/x-git-upload-pack-request")
	req.Header.Add("Accept", "application/x-git-upload-pack-result")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "cannot send a request to the delegate", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	w.Header().Add("Content-Type", "application/x-git-upload-pack-result")
	v1Resp := gitprotocolio.NewProtocolV1UploadPackResponse(resp.Body)
	for v1Resp.Scan() {
		if err := writePacket(w, v1Resp.Chunk()); err != nil {
			writePacket(w, gitprotocolio.ErrorPacket("cannot write a packet"))
			return
		}
	}

	if err := v1Resp.Err(); err != nil {
		if ep, ok := err.(gitprotocolio.ErrorPacket); ok {
			writePacket(w, ep)
		} else {
			writePacket(w, gitprotocolio.ErrorPacket("internal error"))
			log.Printf("Parsing error: %#v, parser: %#v", err, v1Resp)
		}
		return
	}
}

func (s *httpProxyServer) receivePackHandler(w http.ResponseWriter, r *http.Request) {
	u, err := httpURLForReceivePack(s.delegateURL)
	if err != nil {
		http.Error(w, "cannot construct the /git-receive-pack URL", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Content-Encoding") == "gzip" {
		var err error
		if r.Body, err = gzip.NewReader(r.Body); err != nil {
			http.Error(w, "cannot ungzip", http.StatusBadRequest)
			return
		}
	}

	if r.Header.Get("Git-Protocol") == "version=2" {
		serveProtocolV2(u, w, r)
		return
	}
	receivePackV1Handler(u, w, r)
}

func receivePackV1Handler(delegateURL string, w http.ResponseWriter, r *http.Request) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		v1Req := gitprotocolio.NewProtocolV1ReceivePackRequest(r.Body)

		for v1Req.Scan() {
			if err := writePacket(pw, v1Req.Chunk()); err != nil {
				writePacket(pw, gitprotocolio.ErrorPacket("cannot write a packet"))
				return
			}
		}

		if err := v1Req.Err(); err != nil {
			if ep, ok := err.(gitprotocolio.ErrorPacket); ok {
				writePacket(pw, ep)
			} else {
				writePacket(pw, gitprotocolio.ErrorPacket("internal error"))
				log.Printf("Parsing error: %#v, parser: %#v", err, v1Req)
			}
			return
		}
	}()

	req, err := http.NewRequest("POST", delegateURL, pr)
	if err != nil {
		http.Error(w, "cannot construct the request object", http.StatusInternalServerError)
		return
	}
	req.Header.Add("Content-Type", "application/x-git-receive-pack-request")
	req.Header.Add("Accept", "application/x-git-receive-pack-result")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "cannot send a request to the delegate", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	pktWt := synchronizedWriter{w: w}
	mainRd, mainWt := io.Pipe()
	go func() {
		defer mainWt.Close()
		sc := gitprotocolio.NewPacketScanner(resp.Body)
	scanner:
		for sc.Scan() {
			switch p := sc.Packet().(type) {
			case gitprotocolio.BytesPacket:
				sp := gitprotocolio.ParseSideBandPacket(p)
				if mp, ok := sp.(gitprotocolio.SideBandMainPacket); ok {
					if _, err := mainWt.Write(mp); err != nil {
						pktWt.closeWithError(err)
						return
					}
				}
				pktWt.writePacket(sp)
			case gitprotocolio.FlushPacket:
				break scanner
			default:
				pktWt.closeWithError(fmt.Errorf("unexpected packet: %#v", sc.Packet()))
				return
			}
		}
		if err := sc.Err(); err != nil {
			pktWt.closeWithError(err)
		}
	}()
	ch, chunkWt := gitprotocolio.NewChunkedWriter(0xFFFF - 5)
	go func() {
		defer chunkWt.Close()
		v1Resp := gitprotocolio.NewProtocolV1ReceivePackResponse(mainRd)
		for v1Resp.Scan() {
			if err := writePacket(chunkWt, v1Resp.Chunk()); err != nil {
				pktWt.closeWithError(err)
				return
			}
		}
		if err := v1Resp.Err(); err != nil {
			log.Println(err)
			pktWt.closeWithError(err)
		}
	}()

	w.Header().Add("Content-Type", "application/x-git-receive-pack-result")
	for bs := range ch {
		pktWt.writePacket(gitprotocolio.SideBandMainPacket(bs))
	}
	pktWt.writePacket(gitprotocolio.FlushPacket{})

}

func serveProtocolV2(delegateURL string, w http.ResponseWriter, r *http.Request) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		v2Req := gitprotocolio.NewProtocolV2Request(r.Body)

		for v2Req.Scan() {
			if err := writePacket(pw, v2Req.Chunk()); err != nil {
				writePacket(pw, gitprotocolio.ErrorPacket("cannot write a packet"))
				return
			}
		}

		if err := v2Req.Err(); err != nil {
			if ep, ok := err.(gitprotocolio.ErrorPacket); ok {
				writePacket(pw, ep)
			} else {
				writePacket(pw, gitprotocolio.ErrorPacket("internal error"))
				log.Printf("Parsing error: %#v, parser: %#v", err, v2Req)
			}
			return
		}
	}()

	req, err := http.NewRequest("POST", delegateURL, pr)
	if err != nil {
		http.Error(w, "cannot construct the request object", http.StatusInternalServerError)
		return
	}
	req.Header.Add("Content-Type", "application/x-git-upload-pack-request")
	req.Header.Add("Accept", "application/x-git-upload-pack-result")
	req.Header.Add("Git-Protocol", "version=2")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "cannot send a request to the delegate", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	w.Header().Add("Content-Type", "application/x-git-upload-pack-result")
	v2Resp := gitprotocolio.NewProtocolV2Response(resp.Body)
	for v2Resp.Scan() {
		if err := writePacket(w, v2Resp.Chunk()); err != nil {
			writePacket(w, gitprotocolio.ErrorPacket("cannot write a packet"))
			return
		}
	}

	if err := v2Resp.Err(); err != nil {
		if ep, ok := err.(gitprotocolio.ErrorPacket); ok {
			writePacket(w, ep)
		} else {
			writePacket(w, gitprotocolio.ErrorPacket("internal error"))
			log.Printf("Parsing error: %#v, parser: %#v", err, v2Resp)
		}
	}
}

func httpURLForLsRemote(base, service string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Add("service", service)
	u.Path = path.Join(u.Path, "/info/refs")
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func httpURLForUploadPack(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "/git-upload-pack")
	return u.String(), nil
}

func httpURLForReceivePack(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "/git-receive-pack")
	return u.String(), nil
}

func writePacket(w io.Writer, p gitprotocolio.Packet) error {
	_, err := w.Write(p.EncodeToPktLine())
	return err
}

type synchronizedWriter struct {
	w      io.Writer
	m      sync.Mutex
	closed bool
}

func (s *synchronizedWriter) writePacket(p gitprotocolio.Packet) error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.closed {
		return errors.New("already closed")
	}
	_, err := s.w.Write(p.EncodeToPktLine())
	return err
}

func (s *synchronizedWriter) closeWithError(err error) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.w.Write(gitprotocolio.SideBandErrorPacket(err.Error()).EncodeToPktLine())
}
