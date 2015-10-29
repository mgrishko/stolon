// Copyright 2015 Sorint.lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sorintlab/stolon/common"
	"github.com/sorintlab/stolon/pkg/cluster"
	etcdm "github.com/sorintlab/stolon/pkg/etcd"

	"github.com/sorintlab/stolon/Godeps/_workspace/src/github.com/spf13/cobra"
)

var cmdConfigReplace = &cobra.Command{
	Use:   "replace",
	Run:   configReplace,
	Short: "replace configuration",
}

type configReplaceOpts struct {
	file string
}

var crOpts configReplaceOpts

func init() {
	cmdConfigReplace.PersistentFlags().StringVarP(&crOpts.file, "file", "f", "", "file contaning the configuration to replace")

	cmdConfig.AddCommand(cmdConfigReplace)
}

func replaceConfig(e *etcdm.EtcdManager, nc *cluster.NilConfig) error {
	lsi, _, err := e.GetLeaderSentinelInfo()
	if lsi == nil {
		return fmt.Errorf("leader sentinel info not available")
	}

	ncj, err := json.Marshal(nc)
	if err != nil {
		return fmt.Errorf("failed to marshall config: %v", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s:%s/config/current", lsi.ListenAddress, lsi.Port), bytes.NewReader(ncj))
	if err != nil {
		return fmt.Errorf("cannot create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error setting config: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error setting config: leader sentinel returned non ok code: %s", res.Status)
	}
	return nil
}

func configReplace(cmd *cobra.Command, args []string) {
	if crOpts.file == "" {
		die("no config file provided (--file/-f option)")
	}

	config := []byte{}
	var err error
	if crOpts.file == "-" {
		config, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			die("cannot read config file from stdin: %v", err)
		}
	} else {
		config, err = ioutil.ReadFile(crOpts.file)
		if err != nil {
			die("cannot read provided config file: %v", err)
		}
	}

	etcdPath := filepath.Join(common.EtcdBasePath, cfg.clusterName)
	e, err := etcdm.NewEtcdManager(cfg.etcdEndpoints, etcdPath, common.DefaultEtcdRequestTimeout)
	if err != nil {
		die("error: %v", err)
	}

	var nc cluster.NilConfig
	err = json.Unmarshal(config, &nc)
	if err != nil {
		die("failed to marshal config: %v", err)
	}

	if err = replaceConfig(e, &nc); err != nil {
		die("error: %v", err)
	}
}