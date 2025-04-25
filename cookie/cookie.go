package cookie

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

var Map = make(map[string]any)
var RWMutex sync.RWMutex

const filename = "cookie.yaml"

func Initialize() {
	var data, err = os.ReadFile(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}

	err = yaml.Unmarshal(data, &Map)
	if err != nil {
		return
	}
}

func Set(varName string, data any) error {
	RWMutex.Lock()
	defer RWMutex.Unlock()

	Map[varName] = data

	var out, err = yaml.Marshal(Map)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, out, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func Get(varName string) any {
	return Map[varName]
}
