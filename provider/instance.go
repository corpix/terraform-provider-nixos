package provider

import (
	"context"

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

func (i Instance) Create(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var (
		provider          = meta.(*Provider)
		derivationsSchema []map[string]interface{}
		err               error
	)

	derivations, err := provider.Build(ctx, resource)
	if err != nil {
		goto fail
	}

	resource.SetId(derivations.Hash())

	err = provider.Push(ctx, resource, derivations)
	if err != nil {
		goto fail
	}

	derivationsSchema = make([]map[string]interface{}, len(derivations))
	err = mapstructure.Decode(derivations, &derivationsSchema)
	if err != nil {
		goto fail
	}

	err = provider.Switch(ctx, resource, derivations)
	if err != nil {
		goto fail
	}

	//

	err = resource.Set(KeyDerivations, derivationsSchema)
	if err != nil {
		goto fail
	}

	return nil

fail:
	return i.fail(err)
}

func (i Instance) Read(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func (i Instance) Update(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func (i Instance) Delete(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	resource.SetId("")
	return nil
}

var instance Instance
