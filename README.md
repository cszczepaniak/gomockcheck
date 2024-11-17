# `gomockcheck`
`gomockcheck` checks for common problems when using the [testify mock
package](https://github.com/stretchr/testify/?tab=readme-ov-file#mock-package).

### `assertexpectations`
This check enforces that `AssertExpectations` is called as the first usage of a newly constructed
mock object, either in a `defer` or a `t.Cleanup`.
