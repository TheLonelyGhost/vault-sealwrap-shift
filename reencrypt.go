package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	pkcs11 "github.com/hashicorp/go-kms-wrapping-enterprise/wrappers/pkcs11"
	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	alicloudkms "github.com/hashicorp/go-kms-wrapping/wrappers/alicloudkms/v2"
	awskms "github.com/hashicorp/go-kms-wrapping/wrappers/awskms/v2"
	azurekeyvault "github.com/hashicorp/go-kms-wrapping/wrappers/azurekeyvault/v2"
	gcpckms "github.com/hashicorp/go-kms-wrapping/wrappers/gcpckms/v2"
	huaweicloudkms "github.com/hashicorp/go-kms-wrapping/wrappers/huaweicloudkms/v2"
	ibmkp "github.com/hashicorp/go-kms-wrapping/wrappers/ibmkp/v2"
	ocikms "github.com/hashicorp/go-kms-wrapping/wrappers/ocikms/v2"
	tencentcloudkms "github.com/hashicorp/go-kms-wrapping/wrappers/tencentcloudkms/v2"
	transit "github.com/hashicorp/go-kms-wrapping/wrappers/transit/v2"
)

func setupSealWrapper(ctx context.Context, sealType wrapping.WrapperType, sealConfigFile string) (wrapper wrapping.Wrapper, err error) {
	var sealConfig map[string]string
	// NOTE: Takes JSON-encoded object according to documented parameters on each seal mechanism's page
	// See HSM as an example: https://developer.hashicorp.com/vault/docs/configuration/seal/pkcs11#pkcs11-parameters

	switch sealType {
	case "alicloudkms":
		wrapper = alicloudkms.NewWrapper()
	case "awskms":
		wrapper = awskms.NewWrapper()
	case "azurekeyvault":
		wrapper = azurekeyvault.NewWrapper()
	case "gcpckms":
		wrapper = gcpckms.NewWrapper()
	case "huaweicloudkms":
		wrapper = huaweicloudkms.NewWrapper()
	case "ibmkp":
		wrapper = ibmkp.NewWrapper()
	case "ocikms":
		wrapper = ocikms.NewWrapper()
	case "tencentcloudkms":
		wrapper = tencentcloudkms.NewWrapper()
	case "transit":
		wrapper = transit.NewWrapper()
	case "pkcs11":
		wrapper, err = pkcs11.NewWrapper()
	default:
		err = fmt.Errorf("unsupported seal wrapper type: %s", sealType)
	}
	if err != nil {
		return
	}

	// Begin configuring wrapper
	configBytes, err := os.ReadFile(sealConfigFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(configBytes, &sealConfig)
	if err != nil {
		return
	}

	_, err = wrapper.SetConfig(ctx, wrapping.WithConfigMap(sealConfig))
	if err != nil {
		return
	}
	// Wrapper has been configured

	// Begin verifying wrapper
	var blobInfo *wrapping.BlobInfo
	blobInfo, err = wrapper.Encrypt(ctx, []byte("test"))
	if err != nil {
		return
	}
	_, err = wrapper.Decrypt(ctx, blobInfo)
	if err != nil {
		return
	}
	// Verified wrapper works as expected

	return
}
