// Package manifest defines the model manifest: the versioned JSON document an
// adapter prints to describe a model. Names are physical, exactly as they
// exist in the database; the core applies no ORM naming rules.
//
// Parse is the single entry point for turning bytes into a Manifest. Every
// source of manifests (a file, an adapter subprocess, a library caller) goes
// through it so they all get the same validation.
package manifest

import (
	"encoding/json"
	"fmt"
)

// Version is the manifest version this build understands.
const Version = 1

// Manifest describes a model.
type Manifest struct {
	Version int     `json:"version"`
	Tables  []Table `json:"tables"`
}

// Parse decodes and validates a manifest document.
func Parse(data []byte) (*Manifest, error) {
	var head struct {
		Version json.Number `json:"version"`
	}
	if err := json.Unmarshal(data, &head); err == nil && head.Version != "" {
		if v, err := head.Version.Int64(); err == nil && v != Version {
			return nil, fmt.Errorf("manifest version %d is not supported (supported: %d)", v, Version)
		}
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	return &m, nil
}

// Table is a model entity mapped to a database table.
type Table struct{}
