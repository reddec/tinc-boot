package config

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"io"
	"reflect"
	"strings"
)

const (
	blobBegin = "-----BEGIN "
	blobEnd   = "-----END"
	tag       = "tinc"
)

type Scanner interface {
	Scan(value string) error
}

func Unmarshal(data []byte, target interface{}) error {
	return UnmarshalStream(bytes.NewReader(data), target)
}

// Unmarshal TINC config file. Target should be ref to structure.
//
// Names should match fields. If target value is not primitive or slice, it should implement Scanner interface
func UnmarshalStream(reader io.Reader, target interface{}) error {
	val := reflect.ValueOf(target)
	tp := val.Type()
	if tp.Kind() != reflect.Ptr || tp.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("pointer to struct required")
	}
	val = val.Elem()
	scanner := bufio.NewScanner(reader)

	var lineIdx int
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, blobBegin) {
			blobName, blobContent := parseBlob(line, scanner, &lineIdx)
			if field := findFieldByNameOrTag(val, blobName); field.IsValid() {
				err := parseValue(blobContent, field)
				if err != nil {
					return fmt.Errorf("line %d (blob %s): %w", lineIdx+1, blobName, err)
				}
			}
		} else if err := parseLine(line, val); err != nil {
			return fmt.Errorf("line %d (%s): %w", lineIdx+1, line, err)
		}
		lineIdx++
	}
	return nil
}

func parseBlob(line string, scanner *bufio.Scanner, lineCounter *int) (string, string) {
	blobName := line[len(blobBegin):]
	end := strings.Index(blobName, "-")
	if end != -1 {
		blobName = blobName[:end]
	}
	var blobContent = []string{line}
	for scanner.Scan() {
		*lineCounter++
		line = scanner.Text()
		blobContent = append(blobContent, line)
		if strings.HasPrefix(line, blobEnd) {
			break
		}
	}
	return blobName, strings.Join(blobContent, "\n")
}

func parseLine(line string, targetStruct reflect.Value) error {
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] == '#' {
		return nil
	}
	kv := strings.SplitN(line, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("invalid line (no separator) %s", line)
	}
	key := strings.TrimSpace(kv[0])
	value := strings.TrimSpace(kv[1])
	field := findFieldByNameOrTag(targetStruct, key)
	if !field.IsValid() {
		// no such field
		return nil
	}

	if field.Kind() != reflect.Ptr {
		field = field.Addr()
	} else if field.IsNil() {
		val := reflect.New(field.Type().Elem())
		field.Set(val)
	}

	err := parseValue(value, field)
	if err != nil {
		return fmt.Errorf("scan value %s for field %s: %w", value, key, err)
	}
	return nil
}

func findFieldByNameOrTag(value reflect.Value, name string) reflect.Value {
	n := value.Type().NumField()
	var f reflect.Value
	for i := 0; i < n; i++ {
		field := value.Type().Field(i)
		info := inspectField(field)
		if info.Ignore {
			continue
		}
		if info.Name == name {
			return value.Field(i)
		}
		if strings.EqualFold(field.Name, name) {
			f = value.Field(i)
		}
	}
	return f
}

type fieldInfo struct {
	Name   string
	Ignore bool
	Blob   bool
}

func inspectField(field reflect.StructField) fieldInfo {
	if !ast.IsExported(field.Name) {
		return fieldInfo{Ignore: true, Name: field.Name}
	}
	tags, ok := field.Tag.Lookup(tag)
	if !ok {
		return fieldInfo{
			Name:   field.Name,
			Ignore: false,
			Blob:   false,
		}
	}
	var info fieldInfo
	info.Name = field.Name
	nameOpts := strings.Split(tags, ",")

	altName := strings.TrimSpace(nameOpts[0])
	if altName == "-" {
		info.Ignore = true
	} else if altName != "" {
		info.Name = altName
	}
	for _, opt := range nameOpts[1:] {
		if strings.TrimSpace(opt) == "blob" {
			info.Blob = true
		}
	}

	return info
}
