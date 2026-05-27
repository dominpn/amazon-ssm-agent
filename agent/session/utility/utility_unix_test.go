// Package platform contains platform specific utilities.
package utility

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	logmocks "github.com/aws/amazon-ssm-agent/agent/mocks/log"
	"github.com/stretchr/testify/assert"
)

// TestCreateLocalAdminUserConcurrent verifies that concurrent calls to
// CreateLocalAdminUser do not race on user creation.
func TestCreateLocalAdminUserConcurrent(t *testing.T) {
	mockLog := logmocks.NewMockLog()

	// Override lock file to temp path since /var/lib/amazon/ssm/ may not exist in CI
	origLockFile := ssmUserLockFile
	ssmUserLockFile = filepath.Join(os.TempDir(), "ssm-user-create-concurrent-test.lock")
	defer func() {
		os.Remove(ssmUserLockFile)
		ssmUserLockFile = origLockFile
	}()

	// Mock execCommand to simulate useradd behavior
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	var userCreated int32 // 0 = not created, 1 = created

	execCommand = func(name string, args ...string) *exec.Cmd {
		cmdStr := args[len(args)-1] // last arg is the shell command string

		// "id ssm-user" — check if user exists
		if len(cmdStr) > 2 && cmdStr[:2] == "id" {
			if atomic.LoadInt32(&userCreated) == 1 {
				// User exists — return success
				return exec.Command("true")
			}
			// User doesn't exist — return failure
			return exec.Command("false")
		}

		// "useradd -m ssm-user" — create user
		if atomic.CompareAndSwapInt32(&userCreated, 0, 1) {
			return exec.Command("true")
		}
		// Second concurrent useradd — user already exists (exit code 9)
		return exec.Command("sh", "-c", "exit 9")
	}

	// Mock osStat to simulate sudoers file already exists
	origOsStat := osStat
	defer func() { osStat = origOsStat }()
	osStat = func(name string) (os.FileInfo, error) {
		return nil, nil // file exists
	}

	// Run 10 concurrent calls
	var wg sync.WaitGroup
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			u := &SessionUtil{}
			_, errors[idx] = u.CreateLocalAdminUser(mockLog)
		}(i)
	}
	wg.Wait()

	// All calls should succeed (no errors)
	for i, err := range errors {
		assert.Nil(t, err, fmt.Sprintf("goroutine %d returned error: %v", i, err))
	}
}

// TestAcquireUserCreationLock verifies lock acquisition and release
func TestAcquireUserCreationLock(t *testing.T) {
	mockLog := logmocks.NewMockLog()

	// Use a temp file since /var/lib/amazon/ssm/ may not exist in test environments
	origLockFile := ssmUserLockFile
	ssmUserLockFile = filepath.Join(os.TempDir(), "ssm-user-create-test.lock")
	defer func() {
		os.Remove(ssmUserLockFile)
		ssmUserLockFile = origLockFile
	}()

	// Acquire lock
	f, err := acquireUserCreationLock(mockLog)
	assert.Nil(t, err)
	assert.NotNil(t, f)

	// Release lock
	releaseUserCreationLock(mockLog, f)
}

// TestReleaseUserCreationLockNilSafe verifies nil file doesn't panic
func TestReleaseUserCreationLockNilSafe(t *testing.T) {
	mockLog := logmocks.NewMockLog()
	assert.NotPanics(t, func() {
		releaseUserCreationLock(mockLog, nil)
	})
}
