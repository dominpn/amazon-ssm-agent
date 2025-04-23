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
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWriteCloser implements io.WriteCloser for testing
type MockWriteCloser struct {
	mock.Mock
	buffer bytes.Buffer
}

func (m *MockWriteCloser) Write(p []byte) (int, error) {
	args := m.Called(p)
	if args.Error(1) != nil {
		return args.Int(0), args.Error(1)
	}
	n, err := m.buffer.Write(p)
	return n, err
}

func (m *MockWriteCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWriteCloser) GetContent() []byte {
	return m.buffer.Bytes()
}

func TestNewBufferedWriteCloser(t *testing.T) {
	mockWriter := &MockWriteCloser{}

	bwc, err := NewBufferedWriteCloser(mockWriter)

	assert.NoError(t, err)
	assert.NotNil(t, bwc)
	assert.NotNil(t, bwc.bufferedWriter)
	assert.Equal(t, mockWriter, bwc.innerCloser)
	assert.NotNil(t, bwc.closedMtx)
	assert.False(t, bwc.closed)
}

func TestWrite(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		closed      bool
		writeError  error
		expectError bool
		expectedN   int
	}{
		{
			name:        "Successful write",
			data:        []byte("test data"),
			closed:      false,
			writeError:  nil,
			expectError: false,
			expectedN:   9, // len("test data")
		},
		{
			name:        "Write to closed writer",
			data:        []byte("test data"),
			closed:      true,
			writeError:  nil,
			expectError: true,
			expectedN:   0,
		},
		{
			name:        "Write error",
			data:        []byte("test data"),
			closed:      false,
			writeError:  errors.New("write error"),
			expectError: true,
			expectedN:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWriter := &MockWriteCloser{}

			if tc.writeError != nil {
				mockWriter.On("Write", mock.Anything).Return(0, tc.writeError)
			} else {
				mockWriter.On("Write", mock.Anything).Return(len(tc.data), nil)
			}

			bwc, err := NewBufferedWriteCloser(mockWriter)
			assert.NoError(t, err)

			if tc.closed {
				bwc.closed = true
			}

			n, err := bwc.Write(tc.data)

			if tc.expectError {
				if err == nil {
					// due to buffering the write didn't fail. The flush should definitely fail
					err = bwc.Flush()
					assert.Error(t, err)
					assert.Equal(t, len(tc.data), n)
				} else {
					assert.Error(t, err)
					assert.Equal(t, tc.expectedN, n)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedN, n)

				// Flush to ensure data is written to the underlying writer
				err = bwc.Flush()
				assert.NoError(t, err)

				// Verify content
				assert.Equal(t, tc.data, mockWriter.GetContent())
			}
		})
	}
}

func TestFlush(t *testing.T) {
	testCases := []struct {
		name        string
		closed      bool
		expectError bool
	}{
		{
			name:        "Successful flush",
			closed:      false,
			expectError: false,
		},
		{
			name:        "Flush closed writer",
			closed:      true,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWriter := &MockWriteCloser{}
			mockWriter.On("Write", mock.Anything).Return(4, nil)

			bwc, err := NewBufferedWriteCloser(mockWriter)
			assert.NoError(t, err)

			// Write some data
			_, err = bwc.Write([]byte("test"))
			assert.NoError(t, err)

			if tc.closed {
				bwc.closed = true
			}

			err = bwc.Flush()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, []byte("test"), mockWriter.GetContent())
			}
		})
	}
}

func TestConcurrentWriteAndFlush(t *testing.T) {
	testCases := []struct {
		name            string
		numWriters      int
		numFlushers     int
		writeIterations int
		flushIterations int
	}{
		{
			name:            "Multiple writers and flushers",
			numWriters:      5,
			numFlushers:     3,
			writeIterations: 100,
			flushIterations: 50,
		},
		{
			name:            "More writers than flushers",
			numWriters:      10,
			numFlushers:     2,
			writeIterations: 50,
			flushIterations: 25,
		},
		{
			name:            "More flushers than writers",
			numWriters:      2,
			numFlushers:     8,
			writeIterations: 50,
			flushIterations: 25,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWriter := &MockWriteCloser{}
			mockWriter.On("Write", mock.Anything).Return(5, nil)
			mockWriter.On("Close").Return(nil)

			bwc, err := NewBufferedWriteCloser(mockWriter)
			assert.NoError(t, err)

			var wg sync.WaitGroup
			errChan := make(chan error, tc.numWriters+tc.numFlushers)

			// Launch writers
			for i := 0; i < tc.numWriters; i++ {
				wg.Add(1)
				go func(writerID int) {
					defer wg.Done()
					for j := 0; j < tc.writeIterations; j++ {
						data := []byte(fmt.Sprintf("data%d-%d", writerID, j))
						_, err := bwc.Write(data)
						if err != nil {
							errChan <- fmt.Errorf("writer %d failed: %v", writerID, err)
							return
						}
						// Small random sleep to increase chance of race conditions
						time.Sleep(time.Duration(rand.Intn(100)) * time.Microsecond)
					}
				}(i)
			}

			// Launch flushers
			for i := 0; i < tc.numFlushers; i++ {
				wg.Add(1)
				go func(flusherID int) {
					defer wg.Done()
					for j := 0; j < tc.flushIterations; j++ {
						err := bwc.Flush()
						if err != nil {
							errChan <- fmt.Errorf("flusher %d failed: %v", flusherID, err)
							return
						}
						// Small random sleep to increase chance of race conditions
						time.Sleep(time.Duration(rand.Intn(100)) * time.Microsecond)
					}
				}(i)
			}

			// Wait for all goroutines to complete
			wg.Wait()
			close(errChan)

			// Check for any errors
			var errors []error
			for err := range errChan {
				errors = append(errors, err)
			}
			assert.Empty(t, errors, "Expected no errors during concurrent operations, got: %v", errors)

			// Verify final state
			err = bwc.Close()
			assert.NoError(t, err, "Expected successful close after concurrent operations")
		})
	}
}

func TestConcurrentWriteFlushClose(t *testing.T) {
	mockWriter := &MockWriteCloser{}
	mockWriter.On("Write", mock.Anything).Return(5, nil)
	mockWriter.On("Close").Return(nil)

	bwc, err := NewBufferedWriteCloser(mockWriter)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	operations := 100
	errChan := make(chan error, operations*3) // For write, flush, and close operations

	// Launch concurrent writers
	for i := range operations {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("data-%d", id))
			if _, err := bwc.Write(data); err != nil {
				if !errors.Is(err, ErrWriterClosed) { // Ignore expected "writer closed" errors
					errChan <- fmt.Errorf("write error: %v", err)
				}
			}
		}(i)
	}

	// Launch concurrent flushers
	for range operations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := bwc.Flush(); err != nil {
				if !errors.Is(err, ErrWriterClosed) { // Ignore expected "writer closed" errors
					errChan <- fmt.Errorf("flush error: %v", err)
				}
			}
		}()
	}

	// Randomly close the writer while operations are ongoing
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bwc.Close(); err != nil {
			errChan <- fmt.Errorf("close error: %v", err)
		}
	}()

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Check for unexpected errors
	var unexpectedErrors []error
	for err := range errChan {
		if !errors.Is(err, ErrWriterClosed) {
			unexpectedErrors = append(unexpectedErrors, err)
		}
	}
	assert.Empty(t, unexpectedErrors, "Expected no unexpected errors during concurrent operations")
}

func TestRaceFreeBufferedWriter(t *testing.T) {
	if !testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	mockWriter := &MockWriteCloser{}
	mockWriter.On("Write", mock.Anything).Return(5, nil)
	mockWriter.On("Close").Return(nil)

	bwc, err := NewBufferedWriteCloser(mockWriter)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent writes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			bwc.Write([]byte("test data"))
		}
	}()

	// Concurrent flushes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			bwc.Flush()
		}
	}()

	// Concurrent reads of internal state
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = bwc.closed
		}
	}()

	// Wait for all operations to complete
	wg.Wait()

	// Final flush and close
	err = bwc.Flush()
	assert.NoError(t, err)
	err = bwc.Close()
	assert.NoError(t, err)
}

func TestIntegration(t *testing.T) {
	// Test the full lifecycle of the BufferedWriteCloser
	mockWriter := &MockWriteCloser{}
	mockWriter.On("Write", mock.Anything).Return(9, nil)
	mockWriter.On("Close").Return(nil)

	bwc, err := NewBufferedWriteCloser(mockWriter)
	assert.NoError(t, err)

	// Write data
	n, err := bwc.Write([]byte("test data"))
	assert.NoError(t, err)
	assert.Equal(t, 9, n)

	// At this point, data should be buffered but not written
	assert.Equal(t, 0, mockWriter.buffer.Len())
	assert.Equal(t, 9, bwc.bufferedWriter.Buffered())

	// Flush should write the data
	err = bwc.Flush()
	assert.NoError(t, err)
	assert.Equal(t, []byte("test data"), mockWriter.GetContent())

	// Write more data
	n, err = bwc.Write([]byte(" more"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	// Close should flush and close
	err = bwc.Close()
	assert.NoError(t, err)
	assert.Equal(t, []byte("test data more"), mockWriter.GetContent())

	// Verify closed state
	assert.True(t, bwc.closed)

	// Operations after close should fail
	_, err = bwc.Write([]byte("after close"))
	assert.Error(t, err)

	err = bwc.Flush()
	assert.Error(t, err)

	// Second close should be no-op
	err = bwc.Close()
	assert.NoError(t, err)
}
