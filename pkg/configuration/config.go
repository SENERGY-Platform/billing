/*
 *    Copyright 2023 InfAI (CC SES)
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package configuration

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type ConfigStruct struct {
	Job       bool `json:"job"`
	JobMonths int  `json:"job_months"`
	Server    bool `json:"server"`

	ApiPort       string `json:"api_port"`
	CalculatorUrl string `json:"calculator_url"`

	MongoUrl        string `json:"mongo_url"`
	MongoUser       string `json:"mongo_user"`
	MongoPassword   string `json:"mongo_password" config:"secret"`
	MongoAuthSource string `json:"mongo_auth_source"`
	MongoDatabase   string `json:"mongo_database"`
	MongoReplSet    bool   `json:"mongo_repl_set"`
	MongoCollection string `json:"mongo_collection"`

	KeycloakUrl    string `json:"keycloak_url"`
	KeycloakClient string `json:"keycloak_client"`
	KeycloakSecret string `json:"keycloak_secret" config:"secret"`

	Debug      bool   `json:"debug"`
	LogHandler string `json:"log_handler"`
}

type Config = *ConfigStruct

func Load(location string) (config Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		log.Println("error on config load: ", err)
		return config, err
	}
	config = &ConfigStruct{
		MongoUrl:        "mongodb://localhost:27017",
		MongoAuthSource: "admin",
		MongoDatabase:   "billing",
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		log.Println("invalid config json: ", err)
		return config, err
	}
	HandleEnvironmentVars(config)
	return config, nil
}

func isSecret(field reflect.StructField) bool {
	return strings.Contains(field.Tag.Get("config"), "secret")
}

// plainConfig has none of ConfigStruct's methods, so formatting it does not recurse.
type plainConfig ConfigStruct

// masked returns a copy in which every non-empty field tagged config:"secret" is replaced.
func (c ConfigStruct) masked() plainConfig {
	v := reflect.ValueOf(&c).Elem()
	for i := 0; i < v.NumField(); i++ {
		if isSecret(v.Type().Field(i)) && v.Field(i).Kind() == reflect.String && v.Field(i).String() != "" {
			v.Field(i).SetString("***")
		}
	}
	return plainConfig(c)
}

func (c ConfigStruct) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.masked())
}

func (c ConfigStruct) String() string {
	return fmt.Sprintf("%+v", c.masked())
}

func (c ConfigStruct) GoString() string {
	return fmt.Sprintf("%#v", c.masked())
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

// preparations for docker
func HandleEnvironmentVars(config Config) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			if !isSecret(configType.Field(index)) {
				fmt.Println("use environment variable: ", envName, " = ", envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Int {
				i, _ := strconv.Atoi(envValue)
				configValue.FieldByName(fieldName).SetInt(int64(i))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Uint16 {
				i, _ := strconv.ParseUint(envValue, 10, 16)
				configValue.FieldByName(fieldName).SetUint(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, _ := strconv.ParseFloat(envValue, 64)
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
}
