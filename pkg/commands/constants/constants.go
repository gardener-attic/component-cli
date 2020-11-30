// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package constants

import (
	"fmt"
	"os"
	"path/filepath"
)

// CliHomeEnvName is the name of the environment variable that configures the component cli home directory.
const CliHomeEnvName = "COMPONENT_CLI_HOME"

// CliHomeDir returns the home directoy of the components cli.
// It returns the COMPONENT_CLI_HOME if its defined otherwise
// the default "$HOME/.component-cli" is returned.
func CliHomeDir() (string, error) {
	lsHome := os.Getenv(CliHomeEnvName)
	if len(lsHome) != 0 {
		return lsHome, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine the component home directory: %w", err)
	}
	return filepath.Join(homeDir, ".component-cli"), nil
}
