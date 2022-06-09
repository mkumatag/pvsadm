package dhcpserver

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/ppc64le-cloud/pvsadm/pkg"
	"github.com/ppc64le-cloud/pvsadm/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	id string
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get DHCP Server",
	Long:  `Get DHCP Server`,
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

		servers, err := pvmclient.DHCPClient.Get(id)
		if err != nil {
			return fmt.Errorf("failed to get a dhcpserver, err: %v", err)
		}

		spew.Dump(servers)
		return nil
	},
}

func init() {
	getCmd.Flags().StringVar(&id, "id", "", "Instance ID of the Cloud connection")
	_ = getCmd.MarkFlagRequired("id")
}
