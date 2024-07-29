// Package pidfile provides structure and helper functions to create and remove
// PID file. A PID file is usually a file used to store the process ID of a
// running process.
package pidfile // import "github.com/docker/docker/pkg/pidfile"

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"golang.org/x/sys/unix"
)

// Alive returns true if process with a given pid is running. It only considers
// positive PIDs; 0 (all processes in the current process group), -1 (all processes
// with a PID larger than 1), and negative (-n, all processes in process group
// "n") values for pid are never considered to be alive.
func Alive(pid int) bool {
	if pid < 1 {
		return false
	}
	switch runtime.GOOS {
	case "darwin":
		// OS X does not have a proc filesystem. Use kill -0 pid to judge if the
		// process exists. From KILL(2): https://www.freebsd.org/cgi/man.cgi?query=kill&sektion=2&manpath=OpenDarwin+7.2.1
		//
		// Sig may be one of the signals specified in sigaction(2) or it may
		// be 0, in which case error checking is performed but no signal is
		// actually sent. This can be used to check the validity of pid.
		err := unix.Kill(pid, 0)

		// Either the PID was found (no error) or we get an EPERM, which means
		// the PID exists, but we don't have permissions to signal it.
		return err == nil || err == unix.EPERM
	default:
		_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
		return err == nil
	}
}

// Read reads the "PID file" at path, and returns the PID if it contains a
// valid PID of a running process, or 0 otherwise. It returns an error when
// failing to read the file, or if the file doesn't exist, but malformed content
// is ignored. Consumers should therefore check if the returned PID is a non-zero
// value before use.
func Read(path string) (pid int, err error) {
	pidByte, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err = strconv.Atoi(string(bytes.TrimSpace(pidByte)))
	if err != nil {
		return 0, nil
	}
	if pid != 0 && Alive(pid) {
		return pid, nil
	}
	return 0, nil
}

// Write writes a "PID file" at the specified path. It returns an error if the
// file exists and contains a valid PID of a running process, or when failing
// to write the file.
func Write(path string, pid int) error {
	if pid < 1 {
		// We might be running as PID 1 when running docker-in-docker,
		// but 0 or negative PIDs are not acceptable.
		return fmt.Errorf("invalid PID (%d): only positive PIDs are allowed", pid)
	}
	oldPID, err := Read(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if oldPID != 0 {
		return fmt.Errorf("process with PID %d is still running", oldPID)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}
