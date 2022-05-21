package provider

import (
	"context"
	"time"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

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

func (i Instance) Diff(ctx context.Context, data *schema.ResourceDiff, meta interface{}) error {
	provider := meta.(*Provider)

	//

	if data.HasChange(KeyDerivations) {
		data.SetNewComputed(KeyDerivations)
		return nil
	}

	derivationsSchema, ok := data.Get(KeyDerivations).([]interface{})
	if !ok {
		return nil
	}

	oldDerivations, err := i.schemaToDerivations(derivationsSchema)
	if err != nil {
		return err
	}
	newDerivations, err := provider.Build(ctx, data)
	if err != nil {
		return err
	}

	if oldDerivations.Hash() != newDerivations.Hash() {
		data.SetNewComputed(KeyDerivations)
		return nil
	}

	return nil
}

func (i Instance) Create(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*Provider)

	derivations, err := provider.Build(ctx, data)
	if err != nil {
		return i.fail(err)
	}

	//

	retry := provider.Get(KeyRetry).(int)
	retryWait := time.Duration(provider.Get(KeyRetryWait).(int)) * time.Second
	for { // NOTE: terraform retry helpers are utter garbage relying on timeouts, here is more simple implementation
		err = provider.Push(ctx, data, derivations)
		if err != nil {
			if retry > 0 {
				retry--
				// TODO: progressive wait time? (need limit)
				time.Sleep(retryWait)
				continue
			}
			return i.fail(err)
		}
		break
	}

	//

	err = provider.Switch(ctx, data, derivations)
	if err != nil {
		return i.fail(err)
	}

	//

	if data.Id() == "" {
		data.SetId(i.generateId())
	}
	derivationsSchema, err := i.derivationsToSchema(derivations)
	if err != nil {
		return i.fail(err)
	}
	err = data.Set(KeyDerivations, derivationsSchema)
	if err != nil {
		return i.fail(err)
	}

	return nil
}

func (i Instance) Read(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func (i Instance) Update(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return i.Create(ctx, data, meta)
}

func (i Instance) Delete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data.SetId("")
	return nil
}

var instance Instance
