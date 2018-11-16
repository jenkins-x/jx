package util

import (
	"gopkg.in/yaml.v2"
	"strconv"

	"github.com/fatih/structs"
)

// ToStringMapStringFromStruct returns string[map]string from any struct.
// Use structs tag to change map keys. e.g. ServerName string `structs:"server_name"`
func ToStringMapStringFromStruct(obj interface{}) map[string]string {
	conf := structs.Map(obj)
	config := map[string]string{}
	for x, y := range conf {
		switch y.(type) {
		case string:
			config[x] = y.(string)
		case int:
			config[x] = strconv.Itoa(y.(int))
		case int32:
			config[x] = strconv.FormatInt(int64(y.(int32)), 10)
		case int64:
			config[x] = strconv.FormatInt(y.(int64), 10)
		case bool:
			config[x] = strconv.FormatBool(y.(bool))
		case float64:
			config[x] = strconv.FormatFloat(y.(float64), 'f', -1, 64)
		case float32:
			config[x] = strconv.FormatFloat(float64(y.(float32)), 'f', -1, 32)
		case uint:
			config[x] = strconv.FormatUint(uint64(y.(uint)), 10)
		case uint8:
			config[x] = strconv.FormatUint(uint64(y.(uint8)), 10)
		case uint16:
			config[x] = strconv.FormatUint(uint64(y.(uint16)), 10)
		case uint32:
			config[x] = strconv.FormatUint(uint64(y.(uint32)), 10)
		case uint64:
			config[x] = strconv.FormatUint(y.(uint64), 10)
		case []byte:
			// This is also the case for []uint8
			config[x] = string(y.([]byte)[:])
		}
	}
	return config
}

// ToMapStringInterfaceFromStruct marshals a struct to a generic map[string]interface{} by marshalling it to yaml and back
func ToMapStringInterfaceFromStruct(obj interface{}) (map[string]interface{}, error) {
	y, err := yaml.Marshal(&obj)
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{})
	err = yaml.Unmarshal(y, &out)
	return out, err
}

// ToStructFromMapStringInterface marshals a generic map[string]interface{} to a struct by marshalling to yaml and back
func ToStructFromMapStringInterface(m map[string]interface{}, str interface{}) error {
	j, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(j, str)
}
