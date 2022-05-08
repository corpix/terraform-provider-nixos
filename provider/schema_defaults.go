package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type (
	SchemaDefaultFunc    = schema.SchemaDefaultFunc
	SchemaDefaultFuncCtr = func(*schema.Schema) SchemaDefaultFunc
)

func SchemaWithDefaultFuncCtr(ctr SchemaDefaultFuncCtr, s *schema.Schema) *schema.Schema {
	// NOTE: this exists only because Terraform SDK is horribly engineered piece of crap
	// which can not set defaults properly for non primitive data-types
	s.DefaultFunc = ctr(s)
	return s
}

func SchemaMapExtend(original, extension map[string]*schema.Schema) map[string]*schema.Schema {
	m := map[string]*schema.Schema{}
	for k, v := range original {
		m[k] = v
	}
	for k, v := range extension {
		m[k] = v
	}
	return m
}

//

const (
	DefaultUser = "root"
)

//

func DefaultMapFromSchema(s *schema.Schema) SchemaDefaultFunc {
	subschema := s.Elem.(*schema.Resource).Schema
	return func() (interface{}, error) {
		var (
			v   = map[string]interface{}{}
			err error
		)
		for k, schema := range subschema {
			if schema.DefaultFunc != nil {
				v[k], err = schema.DefaultFunc()
				if err != nil {
					return nil, err
				}
			} else {
				v[k] = schema.Default
			}
		}
		return []interface{}{v}, nil
	}
}

//

func DefaultSshConfig() (interface{}, error) {
	return map[string]interface{}{
		"user": DefaultUser,
	}, nil
}

func DefaultAddressFilter() (interface{}, error) {
	return []interface{}{
		"::/0",
		"0.0.0.0/0",
	}, nil
}

func DefaultAddressPriority() (interface{}, error) {
	return map[string]interface{}{
		"0.0.0.0/0": 1,
		"::/0":      0,
		// TODO: we could rearrange if we implement some way to check ipv6 connectivity
		// because sometimes we have ipv6 address, but it is broken for some reason (misconfigured, etc)
	}, nil
}
