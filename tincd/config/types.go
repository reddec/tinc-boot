package config

import (
	"fmt"
	"reflect"
	"strconv"
)

func parseValue(value string, target reflect.Value) error {
	if target.Kind() != reflect.Ptr {
		return parseValue(value, target.Addr())
	}
	switch target.Elem().Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, err := strconv.ParseUint(value, 10, 64); err != nil {
			return err
		} else {
			target.Elem().SetUint(v)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, err := strconv.ParseInt(value, 10, 64); err != nil {
			return err
		} else {
			target.Elem().SetInt(v)
		}
	case reflect.Float32, reflect.Float64:
		if v, err := strconv.ParseFloat(value, 64); err != nil {
			return err
		} else {
			target.Elem().SetFloat(v)
		}
	case reflect.Bool:
		if v, err := strconv.ParseBool(value); err != nil {
			return err
		} else {
			target.Elem().SetBool(v)
		}
	case reflect.String:
		target.Elem().SetString(value)
	case reflect.Slice:
		var subType = target.Elem().Type().Elem()
		if subType.Kind() == reflect.Uint8 {
			// byte array
			target.Elem().SetBytes([]byte(value))
			return nil
		}

		var subTarget = reflect.New(subType)
		err := parseValue(value, subTarget)
		if err != nil {
			return err
		}
		target.Elem().Set(reflect.Append(target.Elem(), subTarget.Elem()))
	case reflect.Struct:
		if v, ok := target.Interface().(Scanner); !ok {
			return fmt.Errorf("should implement Scanner interface")
		} else {
			return v.Scan(value)
		}
	case reflect.Ptr:
		subType := target.Type().Elem().Elem()

		fill := reflect.New(subType)
		target.Elem().Set(fill)
		return parseValue(value, fill)
	default:
		return fmt.Errorf("unknown target kind %v", target.Elem().Kind())
	}
	return nil
}
