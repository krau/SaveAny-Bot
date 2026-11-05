package storage

import (
	"bytes"
	"encoding/json"
	"fmt"

	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

// ValidateJSON rejects unknown or mistyped keys before a storage record is
// persisted. Keeping this check at the boundary makes the stored JSON a
// versioned, typed contract instead of an arbitrary map.
func ValidateJSON(typ string, raw []byte) error {
	st, err := storenum.ParseStorageType(typ)
	if err != nil {
		return err
	}
	cfg, err := newStorageConfig(st, BaseConfig{})
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("invalid %s storage config: %w", st, err)
	}
	return nil
}
