// Copyright 2021 IBM Corp
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dhcpserver

import (
	"fmt"
	"github.com/ppc64le-cloud/pvsadm/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/ppc64le-cloud/pvsadm/pkg"
	"github.com/ppc64le-cloud/pvsadm/pkg/client"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Get PowerVS DHCP servers",
	Long:  `Get PowerVS DHCP servers`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if pkg.Options.InstanceID == "" && pkg.Options.InstanceName == "" {
			return fmt.Errorf("--instance-id or --instance-name required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		opt := pkg.Options

		c, err := client.NewClientWithEnv(opt.APIKey, opt.Environment, opt.Debug)
		if err != nil {
			klog.Errorf("failed to create a session with IBM cloud: %v", err)
			return err
		}

		pvmclient, err := client.NewPVMClientWithEnv(c, opt.InstanceID, opt.InstanceName, opt.Environment)
		if err != nil {
			return err
		}

		dhcpservers, err := pvmclient.DHCPClient.GetAll()
		if err != nil {
			return fmt.Errorf("failed to get the networks, err: %v", err)
		}

		table := utils.NewTable()
		table.Render(dhcpservers, nil)
		return nil
	},
}
