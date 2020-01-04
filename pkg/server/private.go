package server

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/utils"
	"github.com/sirupsen/logrus"
)

func generateTargetList(rw http.ResponseWriter, logger logrus.FieldLogger, cfg *config.Config) {
	err := utils.TemplateExecution(cfg.Templates.TargetList, logger, rw, struct{ Targets []*config.Target }{Targets: cfg.Targets}, 200)
	if err != nil {
		logger.Errorln(err)
		utils.HandleInternalServerError(rw, err, "/", logger, cfg.Templates)
		// Stop here
		return
	}
}
