package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
	)

	err := resource.RetryContext(
		ctx,
		data.Timeout(schema.TimeoutCreate) - time.Minute,
		func () *resource.RetryError {
			derivations, err := provider.Build(ctx, data)
			if err != nil {
				return resource.NonRetryableError(err)
			}

			data.SetId(derivations.Hash())

			err = provider.Push(ctx, data, derivations)
			if err != nil {
				return resource.RetryableError(err)
			}

			derivationsSchema = make([]map[string]interface{}, len(derivations))
			err = mapstructure.Decode(derivations, &derivationsSchema)
			if err != nil {
				return resource.NonRetryableError(err)
			}

			err = provider.Switch(ctx, data, derivations)
			if err != nil {
				return resource.NonRetryableError(err)
			}

			//

			err = data.Set(KeyDerivations, derivationsSchema)
			if err != nil {
				return resource.NonRetryableError(err)
			}

			return nil
		},
	)

	if err != nil {
		return i.fail(err)
	}
	return nil
}

func (i Instance) Read(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func (i Instance) Update(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func (i Instance) Delete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data.SetId("")
	return nil
}

var instance Instance
