// Copyright 2025 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific language governing
// permissions and limitations under the License.
package bufio

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

var ErrWriterClosed = fmt.Errorf("writer is closed")

// BufferedWriteCloser adds buffering and thread-safety to a [io.WriteCloser]
type BufferedWriteCloser struct {
	innerCloser io.Closer
	// bufferedWriterMtx locks the "bufferedWriter" variable
	bufferedWriterMtx *sync.RWMutex
	bufferedWriter    *bufio.Writer
	// closedMtx locks the "closed" variable
	closedMtx *sync.RWMutex
	// closed tells if this BufferedWriteCloser was closed
	closed bool
}

func NewBufferedWriteCloser(w io.WriteCloser) (*BufferedWriteCloser, error) {
	return &BufferedWriteCloser{
		bufferedWriterMtx: &sync.RWMutex{},
		bufferedWriter:    bufio.NewWriter(w),
		innerCloser:       w,
		closedMtx:         &sync.RWMutex{},
		closed:            false,
	}, nil
}

func (bw *BufferedWriteCloser) Write(data []byte) (int, error) {
	bw.closedMtx.RLock()
	defer bw.closedMtx.RUnlock()

	if bw.closed {
		return 0, ErrWriterClosed
	}

	bw.bufferedWriterMtx.Lock()
	defer bw.bufferedWriterMtx.Unlock()
	return bw.bufferedWriter.Write(data)
}

func (bw *BufferedWriteCloser) Flush() error {
	bw.closedMtx.RLock()
	defer bw.closedMtx.RUnlock()

	if bw.closed {
		return ErrWriterClosed
	}

	bw.bufferedWriterMtx.Lock()
	defer bw.bufferedWriterMtx.Unlock()
	return bw.bufferedWriter.Flush()
}

func (bw *BufferedWriteCloser) Close() error {
	bw.closedMtx.Lock()
	defer bw.closedMtx.Unlock()

	if bw.closed {
		return nil
	}

	bw.bufferedWriterMtx.Lock()
	defer bw.bufferedWriterMtx.Unlock()

	// Flush any buffered data
	if err := bw.bufferedWriter.Flush(); err != nil {
		return fmt.Errorf("flush error on close: %v", err)
	}

	// Then close the underlying writer
	if err := bw.innerCloser.Close(); err != nil {
		return fmt.Errorf("underlying writer close error: %v", err)
	}

	bw.closed = true
	return nil
}
