package matrix

import (
	"encoding/gob"
	"fmt"
	"os"

	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func newCryptoStore(accessToken, deviceID, mxid, path string) *cryptoStore {
	store := &cryptoStore{
		AccessToken: accessToken,
		DeviceID:    deviceID,
		UserID:      mxid,
		Pickle:      random.String(32),

		Path: path,

		MemoryStateStore: mautrix.MemoryStateStore{
			Registrations:  make(map[id.UserID]bool),
			Members:        make(map[id.RoomID]map[id.UserID]*event.MemberEventContent),
			MembersFetched: make(map[id.RoomID]bool),
			PowerLevels:    make(map[id.RoomID]*event.PowerLevelsEventContent),
			Encryption:     make(map[id.RoomID]*event.EncryptionEventContent),
			Create:         make(map[id.RoomID]*event.Event),
		},
	}

	store.MemoryStore = crypto.NewMemoryStore(store.saveCallback)

	return store
}

func openCryptoStore(path string) (*cryptoStore, error) {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	var store *cryptoStore

	if err = gob.NewDecoder(file).Decode(store); err != nil {
		return nil, fmt.Errorf("failed to decode file: %w", err)
	}

	return store, nil
}

type cryptoStore struct {
	AccessToken string
	DeviceID    string
	UserID      string
	Pickle      string
	Path        string

	*crypto.MemoryStore //nolint:embeddedstructfieldcheck
	mautrix.MemoryStateStore
}

func (s *cryptoStore) saveCallback() error {
	file, err := os.OpenFile(s.Path, 0o600, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	if err = gob.NewEncoder(file).Encode(s); err != nil {
		return fmt.Errorf("failed to marshal self: %w", err)
	}

	if err = file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}
