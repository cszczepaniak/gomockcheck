package customtype

import "testing"

type MyMockType struct{}

func (m *MyMockType) AssertExpectations(t testing.TB) {}
func (m *MyMockType) On(string, ...any)               {}
func (m *MyMockType) Called(...any)                   {}
