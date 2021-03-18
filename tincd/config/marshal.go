package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
)

// Marshal structure as TINC configuration.
// Custom types should implement Stringer interface
func Marshal(source interface{}) ([]byte, error) {
	var out bytes.Buffer
	err := MarshalStream(&out, source)
	return out.Bytes(), err
}

func MarshalStream(out io.Writer, source interface{}) error {
	writer := bufio.NewWriter(out)
	defer writer.Flush()

	return marshalType(writer, fieldInfo{}, reflect.ValueOf(source), true)
}

func marshalType(out *bufio.Writer, info fieldInfo, value reflect.Value, nested bool) error {
	if value.IsZero() {
		return nil
	}
	switch value.Type().Kind() {
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			if !info.Blob {
				if _, err := out.WriteString(info.Name + " = "); err != nil {
					return err
				}
			}
			// byte array
			if _, err := out.Write(value.Bytes()); err != nil {
				return err
			}
			if err := out.WriteByte('\n'); err != nil {
				return err
			}
			return nil
		}
		num := value.Len()
		for i := 0; i < num; i++ {
			if err := marshalType(out, info, value.Index(i), nested); err != nil {
				return err
			}
		}
	case reflect.Struct:
		if !nested {
			if !info.Blob {
				if _, err := out.WriteString(info.Name + " = "); err != nil {
					return err
				}
			}
			_, err := out.WriteString(fmt.Sprintln(value.Addr().Interface()))
			return err
		}
		n := value.Type().NumField()
		var blobs []int
		for i := 0; i < n; i++ {
			field := value.Type().Field(i)
			info := inspectField(field)
			if info.Ignore {
				continue
			}
			if info.Blob {
				blobs = append(blobs, i)
				continue
			}
			if err := marshalType(out, info, value.Field(i), false); err != nil {
				return err
			}
		}
		if len(blobs) > 0 {
			_, _ = out.WriteString("\n")
		}
		for _, idx := range blobs {
			field := value.Type().Field(idx)
			info := inspectField(field)
			if err := marshalType(out, info, value.Field(idx), false); err != nil {
				return err
			}
		}
	case reflect.Ptr:
		if value.IsNil() {
			return nil
		}
		return marshalType(out, info, value.Elem(), nested)
	case reflect.Bool:
		var strValue = "no"
		if value.Bool() {
			strValue = "yes"
		}
		_, err := out.WriteString(info.Name + " = " + strValue + "\n")
		return err
	default:
		if !info.Blob {
			if _, err := out.WriteString(info.Name + " = "); err != nil {
				return err
			}
		}
		_, err := out.WriteString(fmt.Sprintln(value.Interface()))
		return err
	}
	return nil
}
