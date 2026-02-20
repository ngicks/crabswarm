package model

import (
	"google.golang.org/protobuf/types/known/structpb"
)

func optionalStringFromProto(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func optionalStringToProto(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func optionalBoolFromProto(b *bool) *bool {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

func optionalBoolToProto(b *bool) *bool {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

func optionalInt32FromProto(i *int32) *int32 {
	if i == nil {
		return nil
	}
	v := *i
	return &v
}

func optionalInt32ToProto(i *int32) *int32 {
	if i == nil {
		return nil
	}
	v := *i
	return &v
}

func optionalInt64FromProto(i *int64) *int64 {
	if i == nil {
		return nil
	}
	v := *i
	return &v
}

func optionalInt64ToProto(i *int64) *int64 {
	if i == nil {
		return nil
	}
	v := *i
	return &v
}

func optionalFloat64FromProto(f *float64) *float64 {
	if f == nil {
		return nil
	}
	v := *f
	return &v
}

func optionalFloat64ToProto(f *float64) *float64 {
	if f == nil {
		return nil
	}
	v := *f
	return &v
}

func structFromProto(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func structToProto(m map[string]any) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

func valueFromProto(v *structpb.Value) any {
	if v == nil {
		return nil
	}
	return v.AsInterface()
}

func valueToProto(v any) *structpb.Value {
	if v == nil {
		return nil
	}
	val, err := structpb.NewValue(v)
	if err != nil {
		return nil
	}
	return val
}

func valuesFromProto(vals []*structpb.Value) []any {
	if vals == nil {
		return nil
	}
	result := make([]any, len(vals))
	for i, v := range vals {
		result[i] = valueFromProto(v)
	}
	return result
}

func valuesToProto(vals []any) []*structpb.Value {
	if vals == nil {
		return nil
	}
	result := make([]*structpb.Value, len(vals))
	for i, v := range vals {
		result[i] = valueToProto(v)
	}
	return result
}
