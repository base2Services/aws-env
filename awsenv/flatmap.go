package awsenv

import (
	"fmt"
	"reflect"
	"github.com/kataras/golog"
	"time"
	"math/rand"
	"strings"
	"strconv"
)


func init() {
	rand.Seed(time.Now().UnixNano())
}

// Flatten takes a structure and turns into a flat map[string]string.
//
// Within the "thing" parameter, only primitive values are allowed. Structs are
// not supported. Therefore, it can only be slices, maps, primitives, and
// any combination of those together.
//
// See the tests for examples of what inputs are turned into.
func Flatten(thing map[string]interface{}) map[string]*SsmParameter {
	result := make(map[string]*SsmParameter)

	for k, raw := range thing {
		flatten(result, k, reflect.ValueOf(raw))
	}

	return result
}

func flatten(result map[string]*SsmParameter, prefix string, v reflect.Value) {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			prefix = makePath(prefix)
			result[prefix] = &SsmParameter{ paramName: prefix, paramType: "String", value: "true"}
		} else {
			prefix = makePath(prefix)
			result[prefix] = &SsmParameter{ paramName: prefix, paramType: "String", value: "false"}
		}
	case reflect.Int:
		prefix = makePath(prefix)
		result[prefix] = &SsmParameter{ paramName: prefix, paramType: "String", value: fmt.Sprintf("%d", v.Int())}
	case reflect.Map:
		flattenMap(result, prefix, v)
	case reflect.String:
		prefix = makePath(prefix)
		result[prefix] = &SsmParameter{ paramName: prefix, paramType: "String", value: v.String() }
	default:
		panic(fmt.Sprintf("Unknown: %s", v))
	}
}

func flattenMap(result map[string]*SsmParameter, prefix string, v reflect.Value) {

	for _, k := range v.MapKeys() {
		if k.Kind() == reflect.Interface {
			k = k.Elem()
		}

		paramName := fmt.Sprintf("%s/%s", prefix, k.String())

		if k.Kind() != reflect.String {
			panic(fmt.Sprintf("%s: map key is not string: %s", prefix, k))
		}
		value := v.MapIndex(k)
		if v.MapIndex(k).Elem().Kind() == reflect.Map {
			t := v.MapIndex(k).Elem().MapIndex(reflect.ValueOf("type"))
			if t.IsValid() {
				paramName = makePath(paramName)
				result[paramName] = generateParam(paramName, v.MapIndex(k).Elem())
				continue
			}
		}
		flatten(result, paramName, value)
	}
}

func generateParam(paramName string, param reflect.Value) *SsmParameter {
	golog.Debugf("generate param:%s",paramName)
	ssmParam := SsmParameter{ paramName: paramName, paramType: "String", length: "16"}
	for _, key := range param.MapKeys() {
		fieldKey := fmt.Sprintf("%s",key)
		field := fmt.Sprintf("%v",param.MapIndex(key))
		switch fieldKey {
		case "type":
			if field == "string" {
				ssmParam.paramType = "String"
			} else {
				ssmParam.paramType = "SecureString"
			}
		case "version":
			ssmParam.version = field
		case "length":
			ssmParam.length = field
		case "value":
			ssmParam.value = field
		}
	}

	if ssmParam.value == "" {
		ssmParam.value = generateValue(ssmParam)
	}
	return &ssmParam
}

func generateValue(param SsmParameter) string {
	var value string
	switch param.paramType {
	case "SecureString":
		length, err := strconv.Atoi(param.length)
		if err != nil {
			golog.Fatalf("unable to parse SSM Parameter length:%s", err)
		}
		value = randString(length)
	default:
		value = "1"
	}
	return value
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func makePath(path string) string {
	if strings.Contains(path, "/") {
		return "/" + path
	} else {
		return path
	}
}