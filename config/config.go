package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/visonlv/go-vkit/logger"
	"gopkg.in/yaml.v2"
)

type Config struct {
	m map[string]interface{} // 一维，存储 key 和 value。
	// Client      *Client
	currentPath string
	commonPath  string
}

func read(b []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := yaml.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// ToString casts an interface{} to a string. 在类型明确的情况下推荐使用标准库函数。
func toString(i interface{}) string {
	v := toStringE(i)
	return v
}

// ToStringE casts an interface{} to a string. 在类型明确的情况下推荐使用标准库函数。
func toStringE(i interface{}) string {
	switch s := i.(type) {
	case nil:
		return ""
	case int:
		return strconv.Itoa(s)
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case string:
		return s
	case bool:
		return strconv.FormatBool(s)
	default:
		return "error value"
	}
}

func getStoreValue(i interface{}) interface{} {
	switch s := i.(type) {
	case nil:
		return ""
	case int:
		return s
	case bool:
		return s
	case string:
		return s
	default:
		return "error value"
	}
}

var defaultConfig *Config

func (c *Config) set(key string, val interface{}) error {
	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Map:
		keyMap := make(map[string]interface{})
		for _, k := range v.MapKeys() {
			mapValue := v.MapIndex(k).Interface()
			mapKey := toString(k.Interface())
			keyMap[mapKey] = mapValue
			err := c.set(key+"."+mapKey, mapValue)
			if err != nil {
				return err
			}
		}
		c.m[key] = keyMap
	case reflect.Array, reflect.Slice:
		c.m[key] = val
		for i := 0; i < v.Len(); i++ {
			subKey := fmt.Sprintf("%s[%d]", key, i)
			subValue := v.Index(i).Interface()
			err := c.set(subKey, subValue)
			if err != nil {
				return err
			}
		}
	default:
		c.m[key] = getStoreValue(val)
	}
	return nil
}

func (c *Config) fetchConfig(b []byte) error {
	m, err := read(b)
	if err != nil {
		return err
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		err = c.set(k, m[k])
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) loadSystemEnv() error {
	for _, env := range os.Environ() {
		ss := strings.SplitN(env, "=", 2)
		k := ss[0]
		if len(k) > 0 && len(ss) > 1 {
			v := ss[1]
			c.set(k, v)
		}
	}
	return nil
}

func (c *Config) loadCmdArgs() error {
	for i := 0; i < len(os.Args); i++ {
		s := os.Args[i]
		if strings.HasPrefix(s, "--") {
			ss := strings.SplitN(strings.TrimPrefix(s, "--"), "=", 2)
			k, v := ss[0], ""
			if len(ss) > 1 {
				v = ss[1]
			}
			c.set(k, v)
			continue
		}
	}
	return nil
}

// func SetClient(client *Client) {
// 	c := defaultConfig
// 	c.Client = client
// }

func Get(key string, opts ...interface{}) interface{} {
	c := defaultConfig
	if val, ok := c.m[key]; ok {
		return val
	}
	for _, opt := range opts {
		return opt
	}
	return nil
}

func GetString(key string, opts ...interface{}) string {
	val := Get(key, opts...)
	if val == nil {
		logger.Errorf("GetString key:%s not exit\n", key)
		return ""
	}
	return toString(val)
}

func GetBool(key string, opts ...interface{}) bool {
	val := Get(key, opts...)
	if val == nil {
		logger.Errorf("GetBool key:%s not exit\n", key)
		return false
	}
	switch s := val.(type) {
	case bool:
		return s
	}

	logger.Errorf("GetBool key:%s fail not bool\n", key)
	return false
}

func GetInt(key string, opts ...interface{}) int {
	val := Get(key, opts...)
	if val == nil {
		logger.Errorf("GetInt key:%s not exit\n", key)
		return 0
	}
	switch s := val.(type) {
	case int:
		return s
	}

	logger.Errorf("GetInt key:%s fail not int\n", key)
	return 0
}

func GetMap(key string, opts ...interface{}) map[string]interface{} {
	val := Get(key, opts...)
	if val == nil {
		logger.Errorf("GetMap key:%s not exit\n", key)
		return make(map[string]interface{})
	}
	switch s := val.(type) {
	case map[string]interface{}:
		return s
	}

	logger.Errorf("GetMap key:%s fail not int\n", key)
	return make(map[string]interface{})
}

func GetSlide(key string, opts ...interface{}) []interface{} {
	val := Get(key, opts...)
	if val == nil {
		logger.Errorf("GetSlide key:%s not exit\n", key)
		return make([]interface{}, 0)
	}
	switch s := val.(type) {
	case []interface{}:
		return s
	}

	logger.Errorf("GetSlide key:%s fail not int\n", key)
	return make([]interface{}, 0)
}

func InitConfigs(commonPath string, fileNames ...string) error {
	if defaultConfig != nil {
		logger.Errorf("can not init again")
		return nil
	}
	config := &Config{
		m:           make(map[string]interface{}),
		currentPath: "./config/",
		commonPath:  commonPath,
	}
	config.loadSystemEnv()
	config.loadCmdArgs()
	defaultConfig = config

	c := defaultConfig
	var filePath string
	var content []byte
	for _, fileName := range fileNames {
		//当前conf目录
		//当前公共目录
		if checkFileIsExist(config.currentPath + fileName) {
			filePath = config.currentPath + fileName
		} else if checkFileIsExist(config.commonPath + fileName) {
			filePath = config.commonPath + fileName
		}
		if filePath != "" {
			b, err := ioutil.ReadFile(filePath)
			if err != nil {
				return err
			}
			content = b
		} else {
			//通过网络获取
			// ret, err := c.Client.Load(fileName)
			// if err != nil {
			// 	return err
			// }
			// content = []byte(ret)
			logger.Info("TODO")
		}
		err := c.fetchConfig(content)
		if err != nil {
			return err
		}
	}

	return nil
}
