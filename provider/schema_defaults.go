package provider

import (
	"os/user"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
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

func DefaultUser() (interface{}, error) {
	u, err := user.Current()
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve current operating system username")
	}
	return u.Username, nil
}

func DefaultAddressFilter() (interface{}, error) {
	return []interface{}{
		"::/0",
		"0.0.0.0/0",
	}, nil
}

func DefaultAddressPriority() (interface{}, error) {
	return map[string]interface{}{
		"0.0.0.0/0": 0,
		"::/0":      1,
	}, nil
}
