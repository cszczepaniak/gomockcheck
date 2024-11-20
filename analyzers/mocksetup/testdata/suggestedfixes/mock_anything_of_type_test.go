package suggestedfixes

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	. "github.com/stretchr/testify/mock"
	mockimp "github.com/stretchr/testify/mock"
)

type MyMock struct {
	mock.Mock
}

func (m *MyMock) Method1(r io.Reader, b *strings.Builder) {}

func TestInvalidMockAnythingOfType(t *testing.T) {
	m := &MyMock{}

	// This is okay because io.Reader is an interface. We can't even validate the input argument.
	m.On("Method1", mock.AnythingOfType("foo"), &strings.Builder{})

	// This is not okay because *strings.Builder is concrete, so AnythingOfType is a bit silly since
	// the type system already guarantees that. Should be replaced.
	m.On("Method1", &bytes.Buffer{}, mock.AnythingOfType("*strings.Builder"))                    // want "mock.AnythingOfType is equivalent to mock.Anything when the input type is concrete; use mock.Anything instead"
	m.On("Method1", &bytes.Buffer{}, mockimp.AnythingOfType("not right"))                        // want "mock.AnythingOfType is equivalent to mock.Anything when the input type is concrete; use mock.Anything instead"
	m.On("Method1", &bytes.Buffer{}, AnythingOfType("please don't import things like this tho")) // want "mock.AnythingOfType is equivalent to mock.Anything when the input type is concrete; use mock.Anything instead"
}
