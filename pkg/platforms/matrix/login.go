package matrix

import (
	"context"
	"fmt"
	"os"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
)

func setupClient(cfg map[string]string) (*mautrix.Client, error) {
	client, err := mautrix.NewClient(cfg["homeserver"], "", "")
	if err != nil {
		return nil, fmt.Errorf("matrix: failed to create client: %w", err)
	}

	client.UserAgent = "lightning/" + lightning.VERSION

	_, err = os.Stat(cfg["path"])

	var store *cryptoStore

	if os.IsNotExist(err) {
		_, err = client.Login(context.Background(), &mautrix.ReqLogin{
			Type:             mautrix.AuthTypePassword,
			Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: cfg["username"]},
			Password:         cfg["password"],
			StoreCredentials: true,
		})
		if err != nil {
			return nil, fmt.Errorf("matrix: failed to login: %w", err)
		}

		store = newCryptoStore(client.AccessToken, string(client.DeviceID), string(client.UserID), cfg["store"])
	} else {
		store, err = openCryptoStore(cfg["store"])
		if err != nil {
			return nil, fmt.Errorf("matrix: failed to open store: %w", err)
		}
	}

	if err = setupKeys(cfg, client, store); err != nil {
		return nil, err
	}

	return client, nil
}

func setupKeys(cfg map[string]string, client *mautrix.Client, store *cryptoStore) error {
	client.StateStore = store

	helper, err := cryptohelper.NewCryptoHelper(client, []byte(store.Pickle), &store)
	if err != nil {
		return fmt.Errorf("failed to setup crypto helper: %w", err)
	}

	err = helper.Init(context.Background())
	if err != nil {
		return fmt.Errorf("failed to init crypto helper: %w", err)
	}

	client.Crypto = helper

	keyID, keyData, err := helper.Machine().SSSS.GetDefaultKeyData(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get default key: %w", err)
	}

	key, err := keyData.VerifyRecoveryKey(keyID, cfg["recovery_key"])
	if err != nil {
		return fmt.Errorf("failed to verify recovery key: %w", err)
	}

	err = helper.Machine().FetchCrossSigningKeysFromSSSS(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to fetch cross signing keys: %w", err)
	}

	err = helper.Machine().SignOwnDevice(context.Background(), helper.Machine().OwnIdentity())
	if err != nil {
		return fmt.Errorf("failed to sign own device: %w", err)
	}

	err = helper.Machine().SignOwnMasterKey(context.Background())
	if err != nil {
		return fmt.Errorf("failed to sign own master key: %w", err)
	}

	return nil
}
