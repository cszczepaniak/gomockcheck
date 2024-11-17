package testcode

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

type MyMock struct {
	mock.Mock
}

func (*MyMock) ACall() string {
	return ""
}

func TestSomething(t *testing.T) {
	m1 := &MyMock{}
	// func() {
	// 	m1.AssertExpectations(t)
	// }()
	f := func() {
		m1.AssertExpectations(t)
	}
	t.Cleanup(f)
	_ = *m1
	m1.On("")

	// m1.AssertExpectations(t)
	//
	// foo(t, m1)
	//
}

// func foo(t *testing.T, m1 *MyMock) {
// 	t.Cleanup(func() { m1.AssertExpectations(t) })
// }
