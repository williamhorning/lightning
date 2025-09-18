package matrix

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/id"
)

type matrixConfig struct {
	accessToken string
	deviceID    string
	homeserver  string
	mxid        string
	password    string
	random      string
	recoveryKey string
	username    string
}

func setupClient(cfg matrixConfig) (*mautrix.Client, error) {
	client, err := mautrix.NewClient(cfg.homeserver, id.UserID(cfg.mxid), cfg.accessToken)
	if err != nil {
		return nil, fmt.Errorf("matrix: failed to create client: %w", err)
	}

	client.UserAgent = "lightning/" + lightning.VERSION

	if cfg.accessToken == "" || cfg.deviceID == "" || cfg.mxid == "" {
		_, err = client.Login(context.Background(), &mautrix.ReqLogin{
			Type:             mautrix.AuthTypePassword,
			Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: cfg.username},
			Password:         cfg.password,
			StoreCredentials: true,
		})
		if err != nil {
			return nil, fmt.Errorf("matrix: failed to login: %w", err)
		}

		cfg.deviceID = client.DeviceID.String()
		cfg.accessToken = client.AccessToken
		cfg.mxid = client.UserID.String()

		slog.Info("please set the following in your config:", "device_id", cfg.deviceID,
			"access_token", cfg.accessToken, "mxid", cfg.mxid)
	}

	helper, err := cryptohelper.NewCryptoHelper(
		client,
		[]byte(cfg.random),
		crypto.NewMemoryStore(func() error { return nil }),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup crypto helper: %w", err)
	}

	err = helper.Init(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to init crypto helper: %w", err)
	}

	client.Crypto = helper

	if err = setupKeys(cfg, helper); err != nil {
		return nil, err
	}

	return client, nil
}

func setupKeys(cfg matrixConfig, helper *cryptohelper.CryptoHelper) error {
	keyID, keyData, err := helper.Machine().SSSS.GetDefaultKeyData(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get default key: %w", err)
	}

	key, err := keyData.VerifyRecoveryKey(keyID, cfg.recoveryKey)
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
