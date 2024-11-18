# `gomockcheck`
`gomockcheck` checks for common problems when using the [testify mock
package](https://github.com/stretchr/testify/?tab=readme-ov-file#mock-package).

### `assertexpectations`
This check enforces that `AssertExpectations` is called as the first usage of a newly constructed
mock object, either in a `defer` or a `t.Cleanup`.

### `mocksetup`
This check enforces that mocked function calls are set up correctly. It checks for things like:
- Does the function passed to `mock.On` exist on the thing we're mocking?
- Does the mock setup use the correct number of arguments?
- Do the arguments have the correct types?
