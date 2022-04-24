package provider

import (
	"encoding/json"
	"reflect"
)

type (
	Unmarshaler interface {
		Unmarshal(buf []byte, v interface{}) error
	}
	UnmarshalerPassthrough struct{}
	UnmarshalerJSON        struct{}
)

func (*UnmarshalerPassthrough) Unmarshal(buf []byte, v interface{}) error {
	rv := reflect.ValueOf(v)
	rv.Elem().Set(reflect.ValueOf(buf))
	return nil
}

func (*UnmarshalerJSON) Unmarshal(buf []byte, v interface{}) error {
	return json.Unmarshal(buf, v)
}

//

func NewUnmarshalerPassthrough() *UnmarshalerPassthrough { return &UnmarshalerPassthrough{} }
func NewUnmarshalerJSON() *UnmarshalerJSON               { return &UnmarshalerJSON{} }

var (
	_ Unmarshaler = NewUnmarshalerPassthrough()
	_ Unmarshaler = NewUnmarshalerJSON()
)
