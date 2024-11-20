package testdata

import "testing"

func TestMethodThatDoesExist(t *testing.T) {
	m := &MyMock{}
	m.On("Method1", "").Return(nil).Once()
	m.On("Method2", 1, true, "").Return(nil).Once()
}
