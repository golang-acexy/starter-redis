package redismodule

import (
	"github.com/acexy/golang-toolkit/util"
)

type EncodeDataWrapper struct {
	Value any
}

func (j *EncodeDataWrapper) MarshalBinary() (data []byte, err error) {
	return util.ToJsonBytesError(j.Value)
}

type JsonUnmarshaler struct {
	Value any
}

func (j *JsonUnmarshaler) UnmarshalBinary(data []byte) error {
	return util.ParseJsonError(string(data), j.Value)
}
