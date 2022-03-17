package bucket

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

type targetKeyRewriteData struct {
	Request *http.Request
	User    models.GenericUser
	Target  *config.TargetConfig
	Key     string
}
