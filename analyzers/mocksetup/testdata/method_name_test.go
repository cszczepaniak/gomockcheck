package testdata

import (
	"math/rand/v2"
	"strconv"
	"testing"
)

func TestMethodThatDoesNotExist(t *testing.T) {
	m := &MyMock{}
	m.On("MethodThatDoesNotExist", "").Return(nil).Once() // want `"MethodThatDoesNotExist" is not a method of \*example.com.MyMock`
}

func TestNonConstantMethodName(t *testing.T) {
	m := &MyMock{}
	m.On(randomString(), "").Return(nil).Once() // want "the name of a mocked method should be a constant"
}

func randomString() string {
	return strconv.Itoa(rand.Int())
}

const (
	name1 = "Blah"
	name2 = "BlahBlah"
)

func TestTrickyConstantMethodNameThatDoesNotExist(t *testing.T) {
	m := &MyMock{}
	m.On(name1+name2, "").Return(nil).Once() // want `"BlahBlahBlah" is not a method of \*example.com.MyMock`
}

const (
	name3 = "Metho"
	name4 = "d1"
)

func TestTrickyConstantMethodNameThatDoesExist(t *testing.T) {
	m := &MyMock{}
	m.On(name3+name4, "").Return(nil).Once()
}
