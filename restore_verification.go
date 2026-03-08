package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type restoreVerifiedFile struct {
	Name     string `json:"name"`
	Snapshot string `json:"snapshot"`
	Live     string `json:"live"`
	Match    bool   `json:"match"`
	Error    string `json:"error,omitempty"`
}

type restoreVerificationReport struct {
	Timestamp string                `json:"timestamp"`
	Match     bool                  `json:"match"`
	Complete  bool                  `json:"complete"`
	Files     []restoreVerifiedFile `json:"files"`
}

func verifySnapshotAgainstLive(dataDir string, snapshotFiles []string) error {
	report, err := buildSnapshotVerification(dataDir, snapshotFiles)
	if err != nil {
		return err
	}
	return verificationError(report)
}

func buildSnapshotVerification(dataDir string, snapshotFiles []string) (restoreVerificationReport, error) {
	report := restoreVerificationReport{
		Match:    true,
		Complete: len(snapshotFiles) > 0,
		Files:    make([]restoreVerifiedFile, 0, len(snapshotFiles)),
	}
	var firstErr error
	for _, snapshot := range snapshotFiles {
		live := filepath.Join(dataDir, liveDBBaseName(snapshot))
		fileReport := restoreVerifiedFile{
			Name:     filepath.Base(live),
			Snapshot: snapshot,
			Live:     live,
			Match:    true,
		}
		match, err := filesEqual(snapshot, live)
		if err != nil {
			fileReport.Match = false
			fileReport.Error = err.Error()
			if firstErr == nil {
				firstErr = fmt.Errorf("compare %s with %s: %w", snapshot, live, err)
			}
		} else if !match {
			fileReport.Match = false
		}
		if !fileReport.Match {
			report.Match = false
		}
		report.Files = append(report.Files, fileReport)
	}
	return report, firstErr
}

func verificationError(report restoreVerificationReport) error {
	if !report.Complete {
		return errors.New("snapshot is incomplete")
	}
	if report.Match {
		return nil
	}
	for _, file := range report.Files {
		if file.Error != "" {
			return fmt.Errorf("verification failed for %s: %s", file.Name, file.Error)
		}
		if !file.Match {
			return fmt.Errorf("live file mismatch for %s", file.Name)
		}
	}
	return errors.New("live files do not match snapshot")
}

func liveDBBaseName(snapshotPath string) string {
	base := filepath.Base(snapshotPath)
	switch {
	case strings.HasPrefix(base, "main.sqlite-shm."):
		return "main.sqlite-shm"
	case strings.HasPrefix(base, "main.sqlite-wal."):
		return "main.sqlite-wal"
	default:
		return "main.sqlite"
	}
}

func filesEqual(left, right string) (bool, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return false, err
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}

	lf, err := os.Open(left)
	if err != nil {
		return false, err
	}
	defer lf.Close()

	rf, err := os.Open(right)
	if err != nil {
		return false, err
	}
	defer rf.Close()

	leftBuf := make([]byte, 32*1024)
	rightBuf := make([]byte, 32*1024)
	for {
		leftN, leftErr := lf.Read(leftBuf)
		rightN, rightErr := rf.Read(rightBuf)
		if leftN != rightN {
			return false, nil
		}
		if leftN > 0 && !bytesEqual(leftBuf[:leftN], rightBuf[:rightN]) {
			return false, nil
		}
		if errors.Is(leftErr, io.EOF) && errors.Is(rightErr, io.EOF) {
			return true, nil
		}
		if leftErr != nil && !errors.Is(leftErr, io.EOF) {
			return false, leftErr
		}
		if rightErr != nil && !errors.Is(rightErr, io.EOF) {
			return false, rightErr
		}
	}
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

type liveFileState struct {
	Name    string
	Size    int64
	ModTime int64
}

func captureLiveFileState(dataDir string) ([]liveFileState, error) {
	states := make([]liveFileState, 0, 3)
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		info, err := os.Stat(filepath.Join(dataDir, base))
		if err != nil {
			return nil, err
		}
		states = append(states, liveFileState{
			Name:    base,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
		})
	}
	return states, nil
}

func liveFileStatesEqual(left, right []liveFileState) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
