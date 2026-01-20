package snapshot

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var funcCount = make(map[string]int)

// ValidateSnapshot performs snapshot testing
func ValidateSnapshot(t *testing.T, obj interface{}, depth int, msgAndArgs ...interface{}) {
	skip := 1 + depth

	pc, _, _, _ := runtime.Caller(skip)
	funcName := filepath.Base(runtime.FuncForPC(pc).Name())

	call := funcCount[funcName]
	funcCount[funcName] = call + 1

	filename := filepath.Join("testdata", fmt.Sprintf("%s-%d.json", funcName, call))

	expects, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			create(filename, obj)
			return
		}

		panic(err)
	}

	t.Helper()
	objJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		panic(err)
	}

	if !assert.Equal(t, strings.Trim(string(expects), "\n"), strings.Trim(string(objJSON), "\n"), msgAndArgs...) {
		t.Logf("snapshot %s", filename)
	}
}

func create(filename string, obj interface{}) {
	logrus.WithField("filename", filename).Info("writing snapshot file")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		panic(err)
	}
}
