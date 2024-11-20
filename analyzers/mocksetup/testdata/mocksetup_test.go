package testdata

import (
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/mock"
)

type MyMock struct {
	mock.Mock
}

func (m *MyMock) Method1(a string) error                        { return nil }
func (m *MyMock) Method2(b int, c bool, d string) (bool, error) { return false, nil }
func (m *MyMock) Method3(a int, b string, c ...bool) {
	m.Called(a, b, c)
}

func TestMethodThatDoesNotExist(t *testing.T) {
	m := &MyMock{}
	m.On("MethodThatDoesNotExist", "").Return(nil).Once() // want `"MethodThatDoesNotExist" is not a method of \*example.com.MyMock`
}

func TestMethodThatDoesExist(t *testing.T) {
	m := &MyMock{}
	m.On("Method1", "").Return(nil).Once()
	m.On("Method2", 1, true, "").Return(nil).Once()
}

func TestMockAnything(t *testing.T) {
	m := &MyMock{}
	m.On("Method1", mock.Anything).Return(nil).Once()
	m.On("Method2", mock.Anything, true, mock.Anything).Return(nil).Once()
	m.On("Method3", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	// Arg counts still apply even with mock.Anything
	m.On("Method1", mock.Anything, mock.Anything).Return(nil).Once()                               // want `call is mocked for 2 arguments, but method "Method1" takes 1`
	m.On("Method2", true, mock.Anything).Return(nil).Once()                                        // want `call is mocked for 2 arguments, but method "Method2" takes 3`
	m.On("Method3", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once() // want `call is mocked for 4 arguments, but method "Method3" takes 3`
}

func TestMethodThatDoesExist_WrongNumberOfArgs(t *testing.T) {
	m := &MyMock{}
	m.On("Method1", "string", 123).Return(nil).Once() // want `call is mocked for 2 arguments, but method "Method1" takes 1`
}

func TestMethodThatDoesExist_WrongArgumentTypes(t *testing.T) {
	m := &MyMock{}
	m.On("Method2", "string", true, 123).Return(nil).Once() // want `parameter indexes \[0, 2\] had incorrect types`
}

func TestMethodThatDoesExist_WrongNumberOfArgs_Variadic(t *testing.T) {
	m := &MyMock{}
	m.On("Method3", 123).Return(nil).Once() // want `call is mocked for 1 arguments, but method "Method3" takes 3`
}

func TestMethodThatDoesExist_WrongArgumentTypes_Variadic(t *testing.T) {
	m := &MyMock{}
	m.On("Method3", 123, "", []int{1, 2, 3}).Return().Once() // want `parameter indexes \[2\] had incorrect types`
}

func TestNonConstantMethodName(t *testing.T) {
	m := &MyMock{}
	m.On(randomString(), "").Return(nil).Once() // want "the name of a mocked method should be a constant"
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

func randomString() string {
	return strconv.Itoa(rand.Int())
}
