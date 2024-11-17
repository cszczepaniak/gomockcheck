package defaulttype

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

type MyMock struct {
	mock.Mock
}

func newMyMock(t testing.TB) *MyMock {
	m := &MyMock{}
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func newMyMock_NoCleanup() *MyMock {
	return &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
}

func newMyMock_NoCleanup_Defer(t testing.TB) *MyMock {
	m := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	defer m.AssertExpectations(t)
	return m
}

func newMyMock_NoCleanup_Complicated(t testing.TB) (*MyMock, *MyMock) {
	m1 := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	defer m1.AssertExpectations(t)

	m2 := &MyMock{}
	t.Cleanup(func() { m2.AssertExpectations(t) })

	m3 := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	m3.On("hey")

	return m2, m1
}

func Test_NoAssertion(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.On("Foo").Return().Once()
}

func Test_NoAssertion_AccessField(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.Mock.On("Foo").Return().Once()
}

func Test_DeferAssert(t *testing.T) {
	a := &MyMock{}
	defer a.AssertExpectations(t)
	a.On("Foo").Return().Once()
}

func Test_DeferAssert_OnField(t *testing.T) {
	a := &MyMock{}
	defer a.Mock.AssertExpectations(t)
	a.On("Foo").Return().Once()
}

func Test_DeferAssert_AfterOtherUsage(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.On("Foo").Return().Once()
	defer a.AssertExpectations(t)
}

func Test_TCleanup(t *testing.T) {
	a := &MyMock{}
	t.Cleanup(func() { a.AssertExpectations(t) })
	a.On("Foo").Return().Once()
}

func Test_TCleanup_OnField(t *testing.T) {
	a := &MyMock{}
	t.Cleanup(func() { a.Mock.AssertExpectations(t) })
	a.On("Foo").Return().Once()
}

func Test_TCleanup_AfterOtherUsage(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.On("Foo").Return().Once()
	t.Cleanup(func() { a.AssertExpectations(t) })
}

func Test_GetMockFromFunction(t *testing.T) {
	a := newMyMock(t)
	a.On("Foo").Return().Once()
}

func Test_NormalCallToAssertExpectations(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.AssertExpectations(t)
}