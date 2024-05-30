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

package _import

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/controller"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev2/controllerv2"
	"github.com/IBM-Cloud/bluemix-go/crn"
	"github.com/IBM-Cloud/bluemix-go/models"
	pmodels "github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/ppc64le-cloud/pvsadm/pkg"
	"github.com/ppc64le-cloud/pvsadm/pkg/client"
	"github.com/ppc64le-cloud/pvsadm/pkg/utils"
)

const (
	serviceCredPrefix = "pvsadm-service-cred"
	imageStateActive  = "active"
	jobStateCompleted = "completed"
	jobStateFailed    = "failed"
)

// Find COSINSTANCE details of the Provided bucket
func findCOSInstanceDetails(resources []models.ServiceInstanceV2, bxCli *client.Client) (string, string, crn.CRN) {
	for _, resource := range resources {
		if resource.Crn.ServiceName == "cloud-object-storage" {
			s3client, err := client.NewS3Client(bxCli, resource.Name, pkg.ImageCMDOptions.Region)
			if err != nil {
				continue
			}
			buckets, err := s3client.S3Session.ListBuckets(nil)
			if err != nil {
				continue
			}
			for _, bucket := range buckets.Buckets {
				if *bucket.Name == pkg.ImageCMDOptions.BucketName {
					return resource.Name, resource.Guid, resource.Crn
				}
			}
		}
	}
	return "", "", crn.CRN{}
}

var Cmd = &cobra.Command{
	Use:   "import",
	Short: "Import the image into PowerVS workpace",
	Long: `Import the image into PowerVS workpace
pvsadm image import --help for information

# Set the API key or feed the --api-key commandline argument
export IBMCLOUD_API_KEY=<IBM_CLOUD_API_KEY>

# To Import the imge across the two different IBM account use accesskey and secretkey options

# To Import the image from public bucket use public-bucket option

Examples:

# import image using default storage type (service credential will be autogenerated)
pvsadm image import -n upstream-core-lon04 -b <BUCKETNAME> --object rhel-83-10032020.ova.gz --pvs-image-name test-image -r <REGION>

# import image using default storage type with specifying the accesskey and secretkey explicitly
pvsadm image import -n upstream-core-lon04 -b <BUCKETNAME> --accesskey <ACCESSKEY> --secretkey <SECRETKEY> --object rhel-83-10032020.ova.gz --pvs-image-name test-image -r <REGION>

# with user provided storage type
pvsadm image import -n upstream-core-lon04 -b <BUCKETNAME> --pvs-storagetype <STORAGETYPE> --object rhel-83-10032020.ova.gz --pvs-image-name test-image -r <REGION>

# If user wants to specify the type of OS
pvsadm image import -n upstream-core-lon04 -b <BUCKETNAME> --object rhel-83-10032020.ova.gz --pvs-image-name test-image -r <REGION>

# import image from a public IBM Cloud Storage bucket
pvsadm image import -n upstream-core-lon04 -b <BUCKETNAME> --object rhel-83-10032020.ova.gz --pvs-image-name test-image -r <REGION> --public-bucket
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if pkg.ImageCMDOptions.WorkspaceID == "" && pkg.ImageCMDOptions.WorkspaceName == "" {
			return fmt.Errorf("--workspace-name or --workspace-id required")
		}

		case1 := pkg.ImageCMDOptions.AccessKey == "" && pkg.ImageCMDOptions.SecretKey != ""
		case2 := pkg.ImageCMDOptions.AccessKey != "" && pkg.ImageCMDOptions.SecretKey == ""

		if case1 || case2 {
			return fmt.Errorf("required both --accesskey and --secretkey values")
		}
		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		opt := pkg.ImageCMDOptions
		apikey := pkg.Options.APIKey
		//validate inputs
		validStorageType := []string{"tier3", "tier1"}

		if !utils.Contains(validStorageType, strings.ToLower(opt.StorageType)) {
			klog.Errorf("provide valid StorageType. Allowable values are [tier1, tier3]")
			os.Exit(1)
		}

		bxCli, err := client.NewClientWithEnv(apikey, pkg.Options.Environment, pkg.Options.Debug)
		if err != nil {
			return err
		}

		//Create AccessKey and SecretKey for the bucket provided if bucket access is private
		if (opt.AccessKey == "" || opt.SecretKey == "") && (!opt.Public) {
			//Find CosInstance of the bucket
			var svcs []models.ServiceInstanceV2
			svcs, err = bxCli.ResourceClientV2.ListInstances(controllerv2.ServiceInstanceQuery{
				Type: "service_instance",
			})
			if err != nil {
				return err
			}
			cosInstanceName, cosID, crn := findCOSInstanceDetails(svcs, bxCli)
			if cosInstanceName == "" {
				return fmt.Errorf("failed to find the COS instance for the bucket mentioned: %s", opt.BucketName)
			}

			keys, err := bxCli.GetResourceKeys(cosID)
			if err != nil {
				return fmt.Errorf("failed to list the service credentials: %v", err)
			}

			var cred map[string]interface{}
			var ok bool
			if len(keys) == 0 {
				if opt.ServiceCredName == "" {
					opt.ServiceCredName = serviceCredPrefix + "-" + cosInstanceName
				}

				// Create the service credential if does not exist
				klog.V(2).Infof("Auto Generating the COS Service credential for importing the image with name: %s", opt.ServiceCredName)
				CreateServiceKeyRequest := controller.CreateServiceKeyRequest{
					Name:       opt.ServiceCredName,
					SourceCRN:  crn,
					Parameters: map[string]interface{}{"HMAC": true},
				}
				newKey, err := bxCli.ResourceServiceKey.CreateKey(CreateServiceKeyRequest)
				if err != nil {
					return err
				}
				cred, ok = newKey.Credentials["cos_hmac_keys"].(map[string]interface{})
			} else {
				// Use the service credential already created
				klog.V(2).Info("Reading the existing service credential")
				cred, ok = keys[0].Credentials["cos_hmac_keys"].(map[string]interface{})
			}
			if !ok {
				return fmt.Errorf("failed to get the accessKey and secretKey from service credential")
			}
			//Assign the Access Key and Secret Key for further operation
			opt.AccessKey = cred["access_key_id"].(string)
			opt.SecretKey = cred["secret_access_key"].(string)
		}

		pvmclient, err := client.NewPVMClientWithEnv(bxCli, opt.WorkspaceID, opt.WorkspaceName, pkg.Options.Environment)
		if err != nil {
			return err
		}
		//By default Bucket Access is private
		bucketAccess := "private"

		if opt.Public {
			bucketAccess = "public"
		}
		klog.Infof("Importing image %s. Please wait...", opt.ImageName)
		jobRef, err := pvmclient.ImgClient.ImportImage(opt.ImageName, opt.ImageFilename, opt.Region,
			opt.AccessKey, opt.SecretKey, opt.BucketName, strings.ToLower(opt.StorageType), bucketAccess)
		if err != nil {
			return err
		}
		start := time.Now()
		err = utils.PollUntil(time.Tick(2*time.Minute), time.After(opt.WatchTimeout), func() (bool, error) {
			job, err := pvmclient.JobClient.Get(*jobRef.ID)
			if err != nil {
				return false, fmt.Errorf("image import job failed to complete, err: %v", err)
			}
			if *job.Status.State == jobStateCompleted {
				klog.Infof("Image imported successfully, took %s", time.Since(start))
				return true, nil
			}
			if *job.Status.State == jobStateFailed {
				return false, fmt.Errorf("image import job failed to complete, err: %v", job.Status.Message)
			}
			klog.Infof("Image import is in-progress, current state: %s", *job.Status.State)
			return false, nil
		})
		if err != nil {
			return err
		}

		var image *pmodels.ImageReference = &pmodels.ImageReference{}
		klog.V(1).Info("Retrieving image details")

		if image.ImageID == nil {
			image, err = pvmclient.ImgClient.GetImageByName(opt.ImageName)
			if err != nil {
				return err
			}
		}

		if !opt.Watch {
			klog.Infof("Image import for %s is currently in %s state, Please check the progress in the IBM cloud UI", *image.Name, *image.State)
			return nil
		}
		klog.Infof("Waiting for image %s to be active. Please wait...", opt.ImageName)
		start = time.Now()
		return utils.PollUntil(time.Tick(10*time.Second), time.After(opt.WatchTimeout), func() (bool, error) {
			img, err := pvmclient.ImgClient.Get(*image.ImageID)
			if err != nil {
				return false, fmt.Errorf("failed to import the image, err: %v\n\nRun the command \"pvsadm get events -i %s\" to get more information about the failure", err, pvmclient.InstanceID)
			}
			if img.State == imageStateActive {
				klog.Infof("Successfully imported the image: %s with ID: %s in %s", *image.Name, *image.ImageID, time.Since(start))
				return true, nil
			}
			klog.Infof("Waiting for image to be active. Current state: %s", img.State)
			return false, nil
		})
	},
}

func init() {
	// TODO pvs-instance-name and pvs-instance-id is deprecated and will be removed in a future release
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.WorkspaceName, "pvs-instance-name", "n", "", "PowerVS Instance name.")
	Cmd.Flags().MarkDeprecated("pvs-instance-name", "pvs-instance-name is deprecated, workspace-name should be used")
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.WorkspaceID, "pvs-instance-id", "i", "", "PowerVS Instance ID.")
	Cmd.Flags().MarkDeprecated("pvs-instance-id", "pvs-instance-id is deprecated, workspace-id should be used")
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.WorkspaceName, "workspace-name", "", "", "PowerVS Workspace name.")
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.WorkspaceID, "workspace-id", "", "", "PowerVS Workspace ID.")
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.BucketName, "bucket", "b", "", "Cloud Object Storage bucket name.")
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.COSInstanceName, "cos-instance-name", "s", "", "Cloud Object Storage instance name.")
	// TODO It's deprecated and will be removed in a future release
	Cmd.Flags().MarkDeprecated("cos-instance-name", "will be removed in a future version.")
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.Region, "bucket-region", "r", "", "Cloud Object Storage bucket location.")
	Cmd.Flags().StringVarP(&pkg.ImageCMDOptions.ImageFilename, "object", "o", "", "Cloud Object Storage object name.")
	Cmd.Flags().StringVar(&pkg.ImageCMDOptions.AccessKey, "accesskey", "", "Cloud Object Storage HMAC access key.")
	Cmd.Flags().StringVar(&pkg.ImageCMDOptions.SecretKey, "secretkey", "", "Cloud Object Storage HMAC secret key.")
	Cmd.Flags().StringVar(&pkg.ImageCMDOptions.ImageName, "pvs-image-name", "", "Name to PowerVS imported image.")
	Cmd.Flags().BoolVarP(&pkg.ImageCMDOptions.Public, "public-bucket", "p", false, "Cloud Object Storage public bucket.")
	Cmd.Flags().BoolVarP(&pkg.ImageCMDOptions.Watch, "watch", "w", false, "After image import watch for image to be published and ready to use")
	Cmd.Flags().DurationVar(&pkg.ImageCMDOptions.WatchTimeout, "watch-timeout", 1*time.Hour, "watch timeout")
	Cmd.Flags().StringVar(&pkg.ImageCMDOptions.StorageType, "pvs-storagetype", "tier3", "PowerVS Storage type, accepted values are [tier1, tier3].")
	Cmd.Flags().StringVar(&pkg.ImageCMDOptions.ServiceCredName, "cos-service-cred", "", "IBM COS Service Credential name to be auto generated(default \""+serviceCredPrefix+"-<COS Name>\")")

	_ = Cmd.MarkFlagRequired("bucket")
	_ = Cmd.MarkFlagRequired("bucket-region")
	_ = Cmd.MarkFlagRequired("pvs-image-name")
	_ = Cmd.MarkFlagRequired("object")
	Cmd.Flags().SortFlags = false
}
