package util

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestSetEnv(t *testing.T) {
	a := assert.New(t)
	_, found := os.LookupEnv("test_foo")

	a.False(found)
	unset1 := SetEnv("test_foo", "bar")
	a.Equal("bar", os.Getenv("test_foo"))

	unset2 := SetEnv("test_foo", "bar2")
	a.Equal("bar2", os.Getenv("test_foo"))
	unset2()
	a.Equal("bar", os.Getenv("test_foo"))
	unset1()

	_, found = os.LookupEnv("test_foo")
	a.False(found)
}
