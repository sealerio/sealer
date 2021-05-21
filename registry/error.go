package registry

import (
	"net/http"

	"github.com/pkg/errors"

	distributionClient "github.com/docker/distribution/registry/client"
)

var ErrManifestNotFound = errors.New("manifest not found")

func handleErrorResponse(response *http.Response) error {
	return distributionClient.HandleErrorResponse(response)
}

func isSuccessResponse(code int) bool {
	return distributionClient.SuccessStatus(code)
}
