package customtype

import (
	"testing"
)

type MyMock struct {
	MyMockType
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
	m3.Called()

	return m2, m1
}

func Test_NoAssertion(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.Called()
}

func Test_NoAssertion_AccessField(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.MyMockType.Called()
}

func Test_DeferAssert(t *testing.T) {
	a := &MyMock{}
	defer a.AssertExpectations(t)
	a.Called()
}

func Test_DeferAssert_OnField(t *testing.T) {
	a := &MyMock{}
	defer a.MyMockType.AssertExpectations(t)
	a.Called()
}

func Test_DeferAssert_AfterOtherUsage(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.Called()
	defer a.AssertExpectations(t)
}

func Test_TCleanup(t *testing.T) {
	a := &MyMock{}
	t.Cleanup(func() { a.AssertExpectations(t) })
	a.Called()
}

func Test_TCleanup_OnField(t *testing.T) {
	a := &MyMock{}
	t.Cleanup(func() { a.MyMockType.AssertExpectations(t) })
	a.Called()
}

func Test_TCleanup_AfterOtherUsage(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.Called()
	t.Cleanup(func() { a.AssertExpectations(t) })
}

func Test_GetMockFromFunction(t *testing.T) {
	a := newMyMock(t)
	a.Called()
}

func Test_NormalCallToAssertExpectations(t *testing.T) {
	a := &MyMock{} // want "mocks must have an AssertExpectations registered in a defer or t.Cleanup"
	a.AssertExpectations(t)
}
