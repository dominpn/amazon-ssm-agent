package emitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/amazon-ssm-agent/agent/appconfig"
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/common/telemetry/emitter/internal/fileutil/bufio"
	"github.com/aws/amazon-ssm-agent/common/telemetry/emitter/internal/fileutil/sizelimitedlockedfile"
)

const (
	preIngestionDirPermission  = 0750
	preIngestionFilePermission = 0640
)

var TelemetryPreIngestionDir = filepath.Join(appconfig.TelemetryDataStorePath, "preingestion")

// Emitter is used to emit telemetry from all the agent processes.
// Each namespace has its own file which is locked before writing.
type Emitter struct {
	log log.T

	// maxFileSize is the maximum file size for each namespace in bytes
	maxFileSize uint64

	// autoCloseDuration is time after which to close the file opened for writing
	// Once a file is opened for writing, it stays open until the specified duration.
	// This is done to avoid having to reopen files when rapidly emitting metrics.
	autoCloseDuration time.Duration
	// autoCloseTimersMtx protects autoCloseTimers
	autoCloseTimersMtx *sync.RWMutex
	// autoCloseTimers is a map that holds namespace -> autoCloseTimer mapping
	autoCloseTimers map[string]*autoCloseTimer
	// autoCloseTimersWg is used to wait for all auto-close timers to close during Close
	autoCloseTimersWg sync.WaitGroup

	// openNamespaceFileMtx protects openNamespaceFiles
	openNamespaceFileMtx *sync.RWMutex
	// openNamespaceFiles holds namespace -> [bufio.BufferedWriteCloser] mapping
	openNamespaceFiles map[string]*bufio.BufferedWriteCloser
}

type autoCloseTimer struct {
	// cancelSignal is used to signal the auto-close timer to stop
	cancelSignal chan bool
	done         chan bool
}

// GetTelemetryFilePath gets the file path which contains the pre-ingestion telemetry for the given namespace.
func GetTelemetryFilePath(namespace string) string {
	return filepath.Join(TelemetryPreIngestionDir, fmt.Sprintf("%s.jsonl", namespace))
}

func NewEmitter(log log.T) (e *Emitter) {
	return &Emitter{
		log:                  log,
		maxFileSize:          200 * 1024,       // 200 KiB
		autoCloseDuration:    time.Second * 20, // auto-close opened files after 20 seconds of inactivity
		autoCloseTimersMtx:   &sync.RWMutex{},
		autoCloseTimers:      make(map[string]*autoCloseTimer),
		autoCloseTimersWg:    sync.WaitGroup{},
		openNamespaceFileMtx: &sync.RWMutex{},
		openNamespaceFiles:   make(map[string]*bufio.BufferedWriteCloser),
	}
}

// openFileIfNeeded opens the file for namespace if needed and restarts its auto-close timer.
func (e *Emitter) openFileIfNeeded(namespace string) error {
	e.openNamespaceFileMtx.Lock()
	defer e.openNamespaceFileMtx.Unlock()

	if _, ok := e.openNamespaceFiles[namespace]; !ok {
		path := GetTelemetryFilePath(namespace)
		var lf *sizelimitedlockedfile.File
		var err error

		parentDir := filepath.Dir(path)
		if _, err = os.Stat(parentDir); err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(parentDir, preIngestionDirPermission); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		lf, err = sizelimitedlockedfile.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, preIngestionFilePermission, e.maxFileSize)
		if err != nil {
			return err
		}
		bufferedWriter, err := bufio.NewBufferedWriteCloser(lf)
		if err != nil {
			return err
		}
		e.openNamespaceFiles[namespace] = bufferedWriter
	}

	e.restartAutoCloseTimer(namespace)

	return nil
}

// restartAutoCloseTimer signals any previous running auto-close timer to stop and starts a new one.
func (e *Emitter) restartAutoCloseTimer(namespace string) {
	e.autoCloseTimersMtx.Lock()
	defer e.autoCloseTimersMtx.Unlock()
	// stop the previous auto-close timer
	if a, ok := e.autoCloseTimers[namespace]; ok {
		close(a.cancelSignal)

		// wait for closing
		<-a.done
	}

	a := &autoCloseTimer{
		cancelSignal: make(chan bool),
		done:         make(chan bool),
	}
	e.autoCloseTimers[namespace] = a

	e.autoCloseTimersWg.Add(1)

	go func() {
		defer e.autoCloseTimersWg.Done()

		// delete the entry from the map if it's the latest
		defer func() {
			e.autoCloseTimersMtx.Lock()
			defer e.autoCloseTimersMtx.Unlock()
			latestTimer := e.autoCloseTimers[namespace]
			if latestTimer == a {
				delete(e.autoCloseTimers, namespace)
			}
		}()

		<-a.done
	}()

	// start the auto-close timer
	go func() {
		defer close(a.done)

		select {
		case <-time.After(e.autoCloseDuration):
			err := e.closeFileForNamespace(namespace)
			if err != nil {
				e.log.Warnf("error when auto-closing file for namespace %s : %v", namespace, err)
			}
		case <-a.cancelSignal:
		}
	}()
}

// closeFileForNamespace closes the file for the namespace if it is open
func (e *Emitter) closeFileForNamespace(namespace string) (err error) {
	e.openNamespaceFileMtx.Lock()
	defer e.openNamespaceFileMtx.Unlock()

	return e.unlockedCloseFileForNamespace(namespace)
}

// unlockedCloseFileForNamespace closes the file for the namespace if it is open.
// It does not lock openNamespaceFileMtx. The caller MUST do it.
func (e *Emitter) unlockedCloseFileForNamespace(namespace string) (err error) {
	if file, ok := e.openNamespaceFiles[namespace]; ok {
		err = file.Close()
		if sizelimitedlockedfile.IsSizeLimitReached(err) {
			err = nil
		}
		if err == nil {
			delete(e.openNamespaceFiles, namespace)
		}
	}
	return err
}

// Emit emits the telemetry message to the corresponding file for the namespace
func (e *Emitter) Emit(namespace string, message Message) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if err := e.openFileIfNeeded(namespace); err != nil {
		return fmt.Errorf("failed to open file for namespace %s: %v", namespace, err)
	}

	messageJson, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error when marshaling the telemetry message: %v", err)
	}
	messageJson = append(messageJson, '\n')

	// write the message to file
	e.openNamespaceFileMtx.RLock()
	defer e.openNamespaceFileMtx.RUnlock()
	if file, ok := e.openNamespaceFiles[namespace]; ok {
		_, err = file.Write(messageJson)
		if err != nil {
			if sizelimitedlockedfile.IsSizeLimitReached(err) {
				return nil
			}
			return fmt.Errorf("error when writing the telemetry message to file: %v", err)
		}
	} else {
		return fmt.Errorf("file for namespace %s not found", namespace)
	}
	return nil
}

// Flush flushes the buffers of all the open namespace files
func (e *Emitter) Flush() error {
	e.openNamespaceFileMtx.RLock()
	defer e.openNamespaceFileMtx.RUnlock()

	flushErrs := make([]error, 0)
	// flush all open files
	for _, file := range e.openNamespaceFiles {
		err := file.Flush()
		if err != nil && !sizelimitedlockedfile.IsSizeLimitReached(err) {
			flushErrs = append(flushErrs, err)
		}
	}

	return errors.Join(flushErrs...)
}

// Close closes all of the open files and auto-close timers
func (e *Emitter) Close() error {
	e.autoCloseTimersMtx.Lock()
	// stop all auto-close timers
	for _, a := range e.autoCloseTimers {
		close(a.cancelSignal)
	}
	clear(e.autoCloseTimers)
	e.autoCloseTimersMtx.Unlock()
	// wait for all auto-close timers to stop
	e.autoCloseTimersWg.Wait()

	e.openNamespaceFileMtx.Lock()
	closeErrs := make([]error, 0, len(e.openNamespaceFiles))
	// close all open files
	for namespace := range e.openNamespaceFiles {
		err := e.unlockedCloseFileForNamespace(namespace)
		closeErrs = append(closeErrs, err)
	}
	e.openNamespaceFileMtx.Unlock()

	return errors.Join(closeErrs...)
}
