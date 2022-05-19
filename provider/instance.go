package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

type Instance struct{}

func (i Instance) fail(err error) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  err.Error(),
	}}
}

func (i Instance) Create(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var (
		provider          = meta.(*Provider)
		derivationsSchema []map[string]interface{}

		retry     = provider.Get(KeyRetry).(int)
		retryWait = time.Duration(provider.Get(KeyRetryWait).(int)) * time.Second
	)

	derivations, err := provider.Build(ctx, data)
	if err != nil {
		return i.fail(err)
	}

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

	derivationsSchema = make([]map[string]interface{}, len(derivations))
	err = mapstructure.Decode(derivations, &derivationsSchema)
	if err != nil {
		return i.fail(err)
	}

	err = provider.Switch(ctx, data, derivations)
	if err != nil {
		return i.fail(err)
	}

	//

	data.SetId(derivations.Hash())
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
