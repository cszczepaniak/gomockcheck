package testdata

import (
	"io"

	"github.com/stretchr/testify/mock"
)

type MyMock struct {
	mock.Mock
}

func (m *MyMock) Method1(a string) error                        { return nil }
func (m *MyMock) Method2(b int, c bool, d string) (bool, error) { return false, nil }
func (m *MyMock) Method3(a int, b internal.SomeType, c ...bool) {}
func (m *MyMock) Method4(r io.Reader)                           {}
