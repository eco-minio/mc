/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/probe"
)

//   Configure minio client
//
//   ----
//   NOTE: that the configure command only writes values to the config file.
//   It does not use any configuration values from the environment variables.
//
//   One needs to edit configuration file manually, this is purposefully done
//   so to avoid taking credentials over cli arguments. It is a security precaution
//   ----
//
var configCmd = cli.Command{
	Name:   "config",
	Usage:  "Add new alias to default configuration file [~/.mc/config.json]",
	Action: mainConfig,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} alias NAME HOSTURL

EXAMPLES:
   1. Add alias URLs.
      $ mc config alias zek https://s3.amazonaws.com/

`,
}

// mainConfig is the handle for "mc config" sub-command
func mainConfig(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	arg := ctx.Args().First()
	tailArgs := ctx.Args().Tail()
	if len(tailArgs) > 2 {
		console.Fatalf("Incorrect number of arguments, please read \"mc config help\". %s", errInvalidArgument{})
	}
	configPath, err := getMcConfigPath()
	fatalIf(err)

	fatalIf(doConfig(arg, tailArgs))
	// upon success
	console.Infoln("Alias written successfully to [" + configPath + "].")
}

// saveConfig writes configuration data in json format to config file.
func saveConfig(arg string, aliases []string) *probe.Error {
	switch arg {
	case "alias":
		config, err := addAlias(aliases)
		if err != nil {
			return err.Trace()
		}
		return writeConfig(config).Trace()
	default:
		return probe.NewError(errInvalidArgument{})
	}
}

// doConfig is the handler for "mc config" sub-command.
func doConfig(arg string, aliases []string) *probe.Error {
	return saveConfig(arg, aliases).Trace()
}

// addAlias - add new aliases
func addAlias(aliases []string) (quick.Config, *probe.Error) {
	if len(aliases) < 2 {
		return nil, probe.NewError(errInvalidArgument{})
	}
	conf := newConfigV1()
	config, err := quick.New(conf)
	if err != nil {
		return nil, err.Trace()
	}
	err = config.Load(mustGetMcConfigPath())
	if err != nil {
		return nil, err.Trace()
	}

	aliasName := aliases[0]
	url := strings.TrimSuffix(aliases[1], "/")
	if strings.HasPrefix(aliasName, "http") {
		return nil, probe.NewError(errInvalidAliasName{alias: aliasName})
	}
	if !strings.HasPrefix(url, "http") {
		return nil, probe.NewError(errInvalidURL{URL: url})
	}
	if isAliasReserved(aliasName) {
		return nil, probe.NewError(errReservedAliasName{alias: aliasName})
	}
	if !isValidAliasName(aliasName) {
		return nil, probe.NewError(errInvalidAliasName{alias: aliasName})
	}
	// convert interface{} back to its original struct
	newConf := config.Data().(*configV1)
	if _, ok := newConf.Aliases[aliasName]; ok {
		return nil, probe.NewError(errAliasExists{alias: aliasName})
	}
	newConf.Aliases[aliasName] = url
	newConfig, err := quick.New(newConf)
	if err != nil {
		return nil, err.Trace()
	}
	return newConfig, nil
}