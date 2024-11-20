package testdata

import "testing"

func TestMethodThatDoesExist_WrongNumberOfArgs(t *testing.T) {
	m := &MyMock{}
	m.On("Method1", "string", 123).Return(nil).Once() // want `call is mocked for 2 arguments, but method "Method1" takes 1`
}

func TestMethodThatDoesExist_WrongNumberOfArgs_Variadic(t *testing.T) {
	m := &MyMock{}
	m.On("Method3", 123).Return(nil).Once() // want `call is mocked for 1 arguments, but method "Method3" takes 3`
}
