package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

type Instance struct{}

func (i Instance) Create(ctx context.Context, resource *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var (
		provider          = meta.(*Provider)
		derivationsSchema []map[string]interface{}
		err               error
	)

	derivations, err := provider.Build(resource)
	if err != nil {
		goto fail
	}

	err = provider.Push(resource, derivations)
	if err != nil {
		goto fail
	}

	derivationsSchema = make([]map[string]interface{}, len(derivations))
	err = mapstructure.Decode(derivations, &derivationsSchema)
	if err != nil {
		goto fail
	}

	err = resource.Set(KeyDerivations, derivationsSchema)
	if err != nil {
		goto fail
	}
	resource.SetId(derivations.Hash())

	return nil

fail:
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  err.Error(),
	}}
}
func (Instance) Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// fmt.Println("read", d.Id())
	return nil
}
func (Instance) Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// fmt.Println("update")
	return nil
}
func (Instance) Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// fmt.Println("delete")
	d.SetId("")
	return nil
}

var instance Instance
