package dhcpserver

import (
	"fmt"
	"github.com/ppc64le-cloud/pvsadm/pkg"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dhcpserver",
	Short: "dhcpserver command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if pkg.Options.InstanceID == "" {
			return fmt.Errorf("--instance-id is required")
		}
		if pkg.Options.APIKey == "" {
			return fmt.Errorf("api-key can't be empty, pass the token via --api-key or set IBMCLOUD_API_KEY environment variable")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(deleteCmd)

	Cmd.PersistentFlags().StringVarP(&pkg.Options.InstanceID, "instance-id", "i", "", "Instance ID of the PowerVS instance")
}
