package test

import (
	"log"

	"github.com/servehub/serve/manifest"
	"github.com/servehub/utils"
)

func init() {
	manifest.PluginRegestry.Add("test.integration", TestIntegration{})
}

type TestIntegration struct{}

func (p TestIntegration) Run(data manifest.Manifest) error {
	if data.GetString("env") != data.GetString("current-env") {
		log.Printf("No integration test found for `%s`.\n", data.GetString("current-env"))
		return nil
	}

	return utils.RunCmd(data.GetString("command"))
}
