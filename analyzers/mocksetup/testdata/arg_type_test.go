package testdata

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"example.com/internal"
	"github.com/stretchr/testify/mock"
)

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

func TestMockMatchedBy(t *testing.T) {
	f1 := func() func(string) bool {
		return func(s string) bool { return false }
	}
	m := &MyMock{}
	m.On(
		"Method1",
		mock.MatchedBy(f1()),
	).Return(nil).Once()
	m.On(
		"Method1",
		mock.MatchedBy(func(s bool) string { return "" }), // want `the argument to mock.MatchedBy must be func\(string\) bool`
	).Return(nil).Once()
	m.On(
		"Method1",
		mock.MatchedBy(func(s bool) bool { return false }), // want `the argument to mock.MatchedBy must be func\(string\) bool`
	).Return(nil).Once()
	m.On(
		"Method1",
		mock.MatchedBy("wrong"), // want `the argument to mock.MatchedBy must be func\(string\) bool`
	).Return(nil).Once()
	m.On(
		"Method1",
		mock.MatchedBy(123), // want `the argument to mock.MatchedBy must be func\(string\) bool`
	).Return(nil).Once()

	m.On(
		"Method2",
		mock.MatchedBy(func(i int) bool { return false }),
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()
	m.On(
		"Method2",
		mock.MatchedBy(func(i int) {}), // want `the argument to mock.MatchedBy must be func\(int\) bool`
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()
	m.On(
		"Method2",
		mock.MatchedBy(func(s string) bool { return false }), // want `the argument to mock.MatchedBy must be func\(int\) bool`
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()
	m.On(
		"Method2",
		mock.MatchedBy("wrong"), // want `the argument to mock.MatchedBy must be func\(int\) bool`
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()
	m.On(
		"Method2",
		mock.MatchedBy(123), // want `the argument to mock.MatchedBy must be func\(int\) bool`
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()
}

func TestMethodThatDoesExist_WrongArgumentTypes(t *testing.T) {
	m := &MyMock{}
	m.On("Method2",
		"string", // want `invalid parameter type in mock setup; string is not assignable to int`
		true,
		123, // want `invalid parameter type in mock setup; int is not assignable to string`
	).Return(nil).Once()
	m.On("Method3",
		1,
		true, // want `invalid parameter type in mock setup; bool is not assignable to example.com/internal.SomeType`
		[]bool{false},
	).Return(nil).Once()
}

func TestMethodWithInterfaceParam_DifferentImplementations(t *testing.T) {
	m := &MyMock{}
	m.On("Method4", nil)
	m.On("Method4", &bytes.Buffer{})
	m.On("Method4", &os.File{})
	m.On("Method4", "foo")              // want "invalid parameter type in mock setup; string is not assignable to io.Reader"
	m.On("Method4", &strings.Builder{}) // want `invalid parameter type in mock setup; \*strings.Builder is not assignable to io.Reader`
}

func TestMethodThatDoesExist_WrongArgumentTypes_Variadic(t *testing.T) {
	m := &MyMock{}
	m.On("Method3",
		123,
		internal.SomeType{},
		[]int{1, 2, 3}, // want `invalid parameter type in mock setup; \[\]int is not assignable to \[\]bool`
	).Return().Once()

	// With variadic function calls, the last argument can be confusing so let's make sure the
	// message is different when we have T but needed []T
	m.On("Method3",
		1,
		internal.SomeType{},
		false, // want `invalid parameter type in mock setup; bool is not assignable to \[\]bool \(hint: last parameter is variadic, make it a slice\)`
	).Return(nil).Once()
}
