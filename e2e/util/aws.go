package util

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// EncryptString encrypts a string using provided KMSKeyID.
func EncryptString(str, keyID, region string) ([]byte, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	svc := kms.NewFromConfig(cfg)
	input := &kms.EncryptInput{
		KeyId:     &keyID,
		Plaintext: []byte(str),
	}
	res, err := svc.Encrypt(ctx, input)
	if err != nil {
		return nil, err
	}
	return res.CiphertextBlob, nil
}
