package config

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var config Config

func setFieldDefault(field reflect.Value, fieldType reflect.StructField) {
	defaultTag, ok := fieldType.Tag.Lookup("default")
	if !ok || !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.String:
		if field.String() == "" {
			field.SetString(defaultTag)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() == 0 {
			intValue, err := strconv.ParseInt(defaultTag, 10, 64)
			if err == nil {
				field.SetInt(intValue)
			}
		}
	case reflect.Bool:
		if !field.Bool() {
			boolValue, err := strconv.ParseBool(defaultTag)
			if err == nil {
				field.SetBool(boolValue)
			}
		}
	case reflect.Slice:
		if field.Len() == 0 {
			input := strings.Trim(defaultTag, "[]")
			parts := strings.Split(input, ",")

			for _, part := range parts {
				part = strings.TrimSpace(part)
				field.Set(reflect.Append(field, reflect.ValueOf(part)))
			}
		}
	}
}

func hasInitTag(fieldType reflect.StructField) bool {
	_, ok := fieldType.Tag.Lookup("init")

	return ok
}

func setDefaultsRecursive(value reflect.Value) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		typ := value.Type().Field(i)

		if field.Kind() == reflect.Struct {
			setDefaultsRecursive(field)
		} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if field.IsNil() && hasInitTag(typ) {
				field.Set(reflect.New(field.Type().Elem()))
			}

			if !field.IsNil() {
				setDefaultsRecursive(field)
			}
		} else {
			setFieldDefault(field, typ)
		}
	}
}

func setDefaults(v interface{}) {
	value := reflect.ValueOf(v)

	if value.Kind() != reflect.Ptr || value.IsNil() {
		panic("SetDefaults: expected a non-nil pointer")
	}

	setDefaultsRecursive(value.Elem())
}

func GetConfig() Config {
	if !reflect.DeepEqual(config, Config{}) {
		return config
	}

	configFile, err := os.Open("config.yml")
	if err != nil {
		fmt.Printf("Error opening config file: %v\n", err)
		panic(err)
	}

	configFileBytes, err := io.ReadAll(configFile)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		panic(err)
	}

	err = yaml.Unmarshal(configFileBytes, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling config file: %v\n", err)
		panic(err)
	}

	err = configFile.Close()
	if err != nil {
		fmt.Printf("Error closing config file: %v\n", err)
		panic(err)
	}

	setDefaults(&config)

	return config
}

func GetSettings() Settings {
	return GetConfig().Settings
}
