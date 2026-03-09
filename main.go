package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/rahadiangg/cert-manager-webhook-huawei/pkg/huaweicloud"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// Initialize logger with configurable log level
	// Set LOG_LEVEL environment variable to: debug, info, warn, error (default: info)
	// Set LOG_FORMAT environment variable to: json, text (default: text)
	huaweicloud.InitLogger()

	huaweicloud.Info("Huawei Cloud webhook starting",
		"group_name", GroupName,
		"log_level", os.Getenv("LOG_LEVEL"),
	)

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&huaweicloud.HuaweiCloudSolver{},
	)
}
