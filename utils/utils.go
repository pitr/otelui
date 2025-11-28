package utils

import (
	"fmt"
	"strings"

	v1 "go.opentelemetry.io/proto/otlp/common/v1"
)

func AnyToString(v *v1.AnyValue) string {
	if v == nil {
		return "(null)"
	}
	switch v := v.Value.(type) {
	case *v1.AnyValue_StringValue:
		return v.StringValue
	case *v1.AnyValue_BoolValue:
		return fmt.Sprintf("%t", v.BoolValue)
	case *v1.AnyValue_IntValue:
		return fmt.Sprintf("%d", v.IntValue)
	case *v1.AnyValue_DoubleValue:
		return fmt.Sprintf("%f", v.DoubleValue)
	case *v1.AnyValue_ArrayValue:
		var buf strings.Builder
		buf.WriteByte('[')
		for i, v := range v.ArrayValue.Values {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(AnyToString(v))
		}
		buf.WriteByte(']')
		return buf.String()
	case *v1.AnyValue_KvlistValue:
		var buf strings.Builder
		buf.WriteByte('{')
		for i, kv := range v.KvlistValue.Values {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('"')
			buf.WriteString(kv.Key)
			buf.WriteString(`":`)
			buf.WriteString(AnyToString(kv.Value))
		}
		buf.WriteByte('}')
		return buf.String()
	case *v1.AnyValue_BytesValue:
		return fmt.Sprintf("%x", v.BytesValue)
	default:
		return fmt.Sprintf("%#v", v)
	}
}
