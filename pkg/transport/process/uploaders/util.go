// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package uploaders

import (
	"fmt"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

func createUploadRef(repoCtx cdv2.OCIRegistryRepository, componentName string, componentVersion string) string {
	uploadTag := componentVersion
	if strings.Contains(componentVersion, ":") {
		uploadTag = "latest"
	}

	return fmt.Sprintf("%s/component-descriptors/%s:%s", repoCtx.BaseURL, componentName, uploadTag)
}
