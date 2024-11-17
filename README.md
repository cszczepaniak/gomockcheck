# `gomockcheck`
`gomockcheck` checks for common problems when using the [testify mock
package](github.com/stretchr/testify/mock).

### `assertexpectations`
This check enforces that `AssertExpectations` is called as the first usage of a newly constructed
mock object, either in a `defer` or a `t.Cleanup`.
