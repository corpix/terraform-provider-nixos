package provider

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"encoding/hex"
	mathRand "math/rand"
	"strconv"
	"time"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

func init() {
	mathRand.Seed(time.Now().UnixNano())
}

type Instance struct{}

func (i Instance) derivationsToSchema(derivations Derivations) ([]interface{}, error) {
	schema := make([]interface{}, len(derivations))
	err := mapstructure.Decode(derivations, &schema)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func (i Instance) schemaToDerivations(schema []interface{}) (Derivations, error) {
	derivations := make(Derivations, len(schema))
	for n, schemaRecord := range schema {
		err := mapstructure.Decode(schemaRecord, &derivations[n])
		if err != nil {
			return nil, err
		}
	}
	return derivations, nil
}

func (i Instance) secretsFingerprintToSchema(secrets SecretsData) (map[string]interface{}, error) {
	saltSize := 32
	minIterations := 32
	maxIterations := 64

	//

	salt := make([]byte, saltSize)
	_, err := cryptoRand.Read(salt)
	if err != nil {
		return nil, err
	}
	kdfIterations := mathRand.Intn((maxIterations - minIterations + 1) + minIterations)
	sum := secrets.Hash(salt, kdfIterations)

	return map[string]interface{}{
		KeySecretsFingerprintSum:           hex.EncodeToString(sum),
		KeySecretsFingerprintSalt:          hex.EncodeToString(salt),
		KeySecretsFingerprintKdfIterations: strconv.Itoa(kdfIterations),
	}, nil
}

func (i Instance) schemaToSecretsFingerprint(schema map[string]interface{}) (
	sumBytes []byte,
	saltBytes []byte,
	kdfIterationsInt int,
	err error,
) {
	sum, ok := schema[KeySecretsFingerprintSum]
	if !ok {
		return
	}
	salt, ok := schema[KeySecretsFingerprintSalt]
	if !ok {
		return
	}
	kdfInterations, ok := schema[KeySecretsFingerprintKdfIterations]
	if !ok {
		return
	}

	//

	sumBytes, err = hex.DecodeString(sum.(string))
	if err != nil {
		return
	}
	saltBytes, err = hex.DecodeString(salt.(string))
	if err != nil {
		return
	}
	kdfIterationsInt, err = strconv.Atoi(kdfInterations.(string))
	if err != nil {
		return
	}

	return
}

func (i Instance) generateId() string {
	out, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return out
}

func (i Instance) fail(err error) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  err.Error(),
	}}
}

//

func (i Instance) Diff(ctx context.Context, resource *schema.ResourceDiff, meta interface{}) error {
	provider := meta.(*Provider)

	//

	if resource.HasChange(KeySecretsFingerprint) {
		resource.SetNewComputed(KeySecretsFingerprint)
	} else {
		fingerprint, ok := resource.Get(KeySecretsFingerprint).(map[string]interface{})
		if ok {
			sum, salt, kdfIterations, err := i.schemaToSecretsFingerprint(fingerprint)
			if err != nil {
				return err
			}

			secrets, err := provider.NewSecrets(resource)
			if err != nil {
				return err
			}
			secretsData, err := secrets.Data()
			if err != nil {
				return err
			}

			if !bytes.Equal(secretsData.Hash(salt, kdfIterations), sum) {
				resource.SetNewComputed(KeySecretsFingerprint)
			}
		}
	}

	//

	if resource.HasChange(KeyDerivations) {
		resource.SetNewComputed(KeyDerivations)
	} else {
		derivationsSchema, ok := resource.Get(KeyDerivations).([]interface{})
		if ok {
			oldDerivations, err := i.schemaToDerivations(derivationsSchema)
			if err != nil {
				return err
			}
			newDerivations, err := provider.Build(ctx, resource)
			if err != nil {
				return err
			}

			if oldDerivations.Hash() != newDerivations.Hash() {
				resource.SetNewComputed(KeyDerivations)
			}
		}
	}

	return nil
}

func (i Instance) Create(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*Provider)

	derivations, err := provider.Build(ctx, resource)
	if err != nil {
		return i.fail(err)
	}

	//

	secrets, err := provider.NewSecrets(resource)
	if err != nil {
		return i.fail(err)
	}
	defer secrets.Close()
	secretsData, err := secrets.Data()
	if err != nil {
		return i.fail(err)
	}

	//

	retry := provider.Get(KeyRetry).(int)
	retryWait := time.Duration(provider.Get(KeyRetryWait).(int)) * time.Second
	for { // NOTE: terraform retry helpers are utter garbage relying on timeouts, here is more simple implementation
		err = provider.CopySecrets(ctx, resource, secrets)
		if err != nil {
			goto retry
		}
		err = provider.Push(ctx, resource, derivations)
		if err != nil {
			goto retry
		}
		break

	retry:
		if retry > 0 {
			retry--
			// TODO: progressive wait time? (need limit)
			time.Sleep(retryWait)
			continue
		}
		return i.fail(err)
	}

	//

	err = provider.Switch(ctx, resource, derivations)
	if err != nil {
		return i.fail(err)
	}

	//

	if resource.Id() == "" {
		resource.SetId(i.generateId())
	}

	secretsFingerprintSchema, err := i.secretsFingerprintToSchema(secretsData)
	if err != nil {
		return i.fail(err)
	}
	derivationsSchema, err := i.derivationsToSchema(derivations)
	if err != nil {
		return i.fail(err)
	}

	err = resource.Set(KeySecretsFingerprint, secretsFingerprintSchema)
	if err != nil {
		return i.fail(err)
	}
	err = resource.Set(KeyDerivations, derivationsSchema)
	if err != nil {
		return i.fail(err)
	}

	return nil
}

func (i Instance) Read(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func (i Instance) Update(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return i.Create(ctx, resource, meta)
}

func (i Instance) Delete(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	resource.SetId("")
	return nil
}

var instance Instance
