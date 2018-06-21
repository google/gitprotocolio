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

package gitprotocolio

import (
	"bytes"
	"io"
)

type WriteFlushCloser interface {
	io.WriteCloser
	Flush() error
}

type chunkedWriter struct {
	buf bytes.Buffer
	sz  int
	ch  chan<- []byte
}

func NewChunkedWriter(sz int) (<-chan []byte, WriteFlushCloser) {
	ch := make(chan []byte)
	return ch, &chunkedWriter{sz: sz, ch: ch}
}

func (w *chunkedWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if err != nil {
		return n, err
	}
	if w.sz <= w.buf.Len() {
		bs := make([]byte, w.sz)
		for w.sz <= w.buf.Len() {
			rdsz, err := w.buf.Read(bs)
			if err != nil {
				return n, err
			}
			w.ch <- bs[:rdsz]
		}
	}
	return n, nil
}

func (w *chunkedWriter) Flush() error {
	bs := make([]byte, w.sz)
	for {
		rdsz, err := w.buf.Read(bs)
		if rdsz != 0 {
			w.ch <- bs[:rdsz]
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (w *chunkedWriter) Close() error {
	if err := w.Flush(); err != nil {
		return err
	}
	close(w.ch)
	return nil
}
