// Package errs holds error classifiers shared across the provider.
package errs

import (
	"errors"
	"net/http"

	"github.com/uptrace/oapi-codegen-dd/v3/pkg/runtime"
)

// IsNotFound reports whether err is an API 404 error.
func IsNotFound(err error) bool {
	var clientErr *runtime.ClientAPIError
	if errors.As(err, &clientErr) {
		return clientErr.StatusCode() == http.StatusNotFound
	}
	return false
}
