/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"os"
)

type ImageFunction func() string

func CraneRunnerImage() string {
	return getEnvVar("RELATED_IMAGE_CRANE_RUNNER", "quay.io/konveyor/crane-runner:latest")
}

func CraneUIPluginImage() string {
	return getEnvVar("RELATED_IMAGE_CRANE_UI_PLUGIN", "quay.io/konveyor/crane-ui-plugin:latest")
}

func CraneReverseProxyImage() string {
	return getEnvVar("RELATED_IMAGE_CRANE_REVERSE_PROXY", "quay.io/konveyor/crane-reverse-proxy:latest")
}

func CraneSecretServiceImage() string {
	return getEnvVar("RELATED_IMAGE_CRANE_SECRET_SERVICE", "quay.io/konveyor/crane-secret-service:latest")
}

func getEnvVar(key, def string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return def
}
