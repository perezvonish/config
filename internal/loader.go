package internal

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func LoadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.Trim(value, `"'`)

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func LoadFromEnv(cfg interface{}) error {
	v := reflect.ValueOf(cfg)

	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a struct")
	}

	return processStruct(v)
}

func processStruct(v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanSet() {
			continue
		}

		if field.Kind() == reflect.Struct {
			if err := processStruct(field); err != nil {
				return err
			}
			continue
		}

		envTag := fieldType.Tag.Get("env")
		defaultTag := fieldType.Tag.Get("envDefault")
		requiredTag := fieldType.Tag.Get("required")

		if envTag == "" {
			continue
		}

		envVal := os.Getenv(envTag)

		if requiredTag == "true" && envVal == "" {
			return NewFieldRequiredError(envTag)
		}

		if envVal == "" {
			envVal = defaultTag
		}

		if err := setField(field, envVal, envTag); err != nil {
			return err
		}
	}

	return nil
}

func setField(field reflect.Value, value string, envName string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			return nil
		}
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int value for %s: %v", envName, err)
		}
		field.SetInt(intVal)

	default:
		return fmt.Errorf("unsupported type %s for field %s", field.Kind(), envName)
	}

	return nil
}

func NewFieldRequiredError(field string) error {
	return fmt.Errorf("%s is required but not set", field)
}
