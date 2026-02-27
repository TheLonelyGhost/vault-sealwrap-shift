package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

type KeyInfo struct {
	KeyID     string `json:"key_id"`
	HMACKeyID string `json:"hmac_key_id"`
}

type ConvertConfig struct {
	OldKey KeyInfo `json:"old"`
	NewKey KeyInfo `json:"new"`
}

func main() {
	vaultDbPath := flag.String("file", "./vault.db", "Path to the Integrated Storage vault.db file")
	migrateConfigPath := flag.String("config", "./migrate.json", "Path to a file containing old and new key mappings")
	flag.Parse()

	if vaultDbPath == nil {
		panic("Must have path to valid vault.db file")
	}
	if migrateConfigPath == nil {
		panic("Must have path to valid migration manifest file")
	}
	var config ConvertConfig

	configBytes, err := os.ReadFile(*migrateConfigPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		panic(err)
	}

	_, err = os.Stat(*vaultDbPath)
	if err != nil {
		panic(err)
	}

	if config.OldKey.KeyID == "" {
		panic("Old encryption key must be set to some valid value (`.old.key_id`)")
	}
	if config.OldKey.HMACKeyID == "" {
		panic("Old HMAC key must be set to some valid value (`.old.hmac_key_id`)")
	}

	if config.NewKey.KeyID == "" {
		panic("New encryption key must be set to some valid value (`.new.key_id`)")
	}
	if config.NewKey.HMACKeyID == "" {
		panic("New HMAC key must be set to some valid value (`.new.hmac_key_id`)")
	}
	convertSealWrap(
		*vaultDbPath,
		config,
	)
}

func convertSealWrap(vaultDbPath string, config ConvertConfig) (err error) {
	db, err := bolt.Open(vaultDbPath, 0o600, &bolt.Options{
		ReadOnly: false,
		Timeout:  2 * time.Second,
	})
	if err != nil {
		return
	}

	err = db.Update(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte("data"))
		if b == nil {
			return errors.New("no such bucket 'data'")
		}
		c := b.Cursor()
		for boltKey, _ := c.First(); boltKey != nil; boltKey, _ = c.Next() {
			result := b.Get([]byte(boltKey))
			blobInfo, err := protoUnmarshal(result)
			if err != nil {
				err = nil
				// fmt.Printf("Skipping key '%s': it is not wrapped in expected protobuf materials\n", boltKey)
				// NOTE: Not all values in Vault's boltdb use the expected protobuf wrapper. We only care about ones that do.
				continue
			}

			if blobInfo.KeyInfo.KeyId == config.OldKey.KeyID && blobInfo.KeyInfo.HmacKeyId == config.OldKey.HMACKeyID {
				fmt.Printf("Updated key info for '%s'\n", boltKey)
				blobInfo.KeyInfo.KeyId = config.NewKey.KeyID
				blobInfo.KeyInfo.HmacKeyId = config.NewKey.HMACKeyID

				data, err := proto.Marshal(blobInfo)
				if err != nil {
					return err
				}
				err = b.Put(boltKey, data)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Skipped updating key info for '%s': precondition failed\n", boltKey)
			}
			out, err := json.MarshalIndent(blobInfo, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
		}

		// TODO: Find ref to seal migration keyring and change it accordingly
		return
	})
	return
}

func protoUnmarshal(data []byte) (*wrapping.BlobInfo, error) {
	blobInfo := &wrapping.BlobInfo{}
	if err := proto.Unmarshal(data, blobInfo); err != nil {
		eLen := len(data)
		if err := proto.Unmarshal(data[:eLen-1], blobInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ciphertext to blob: %s: %v", err, blobInfo)
		}
	}
	return blobInfo, nil
}
