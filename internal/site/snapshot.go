package site

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const snapshotMetadataPath = "_assets/snapshot.json"

type snapshotMetadata struct {
	Snapshot string `json:"snapshot"`
}

func newSnapshotID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func writeSnapshotMetadata(outDir, snapshotID string) error {
	data, err := json.Marshal(snapshotMetadata{Snapshot: snapshotID})
	if err != nil {
		return fmt.Errorf("encode snapshot metadata: %w", err)
	}
	data = append(data, '\n')
	target := filepath.Join(outDir, filepath.FromSlash(snapshotMetadataPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create snapshot metadata directory: %w", err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("write snapshot metadata: %w", err)
	}
	return nil
}
