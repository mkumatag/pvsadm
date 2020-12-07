package prep

import (
	"fmt"
	"github.com/ppc64le-cloud/pvsadm/pkg/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

var (
	hostPartitions = []string{"/proc", "/dev", "/sys", "/var/run/", "/etc/machine-id"}
)

type PrepareOptions struct {
	// Temp mount directory used for the image preparation
	Mount string

	// OS distribution volume
	Volume string

	// Basic setup configuration
	Setup

	// Additional configuration for the image preparation like nameservers, yum repos etc.
	AdditionalSetup
}

//prepare is a function prepares the CentOS or RHEL image for capturing, this includes
// - Installs the cloud-init
// - Install and configure multipath for rootfs
// - Install all the required modules for PowerVM
// - Sets the root password
func prepare(opt PrepareOptions) error {
	lo, err := setupLoop(opt.Volume)
	if err != nil {
		return err
	}

	err = partprobe(lo)
	if err != nil {
		return err
	}

	// TODO: Get this partition number from the image
	partition := "2"
	partDev := lo + "p" + partition

	err = mount("nouuid", partDev, opt.Mount)
	if err != nil {
		return err
	}
	defer Umount(opt.Mount)

	err = growpart(lo, partition)
	if err != nil {
		return err
	}

	fsType, err := getFSType(partDev)
	if err != nil {
		return err
	}

	switch fsType {
	case "xfs":
		err = xfsGrow(partDev)
		if err != nil {
			return err
		}
	case "ext2", "ext3", "ext4":
		err = resize2fs(partDev)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unable to handle the %s filesystem for %s", fsType, partDev)
	}

	// mount the host partitions
	for _, p := range hostPartitions {
		err = mount("bind", p, filepath.Join(opt.Mount, p))
		if err != nil {
			return err
		}
	}
	defer UmountHostPartitions(opt.Mount)

	setupStr, err := Render(opt)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(opt.Mount, "setup.sh"), []byte(setupStr), 744)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(opt.Mount, "/etc/cloud/cloud.cfg"), []byte(cloudConfig), 644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(opt.Mount, "/etc/cloud/ds-identify.conf"), []byte(dsIdentify), 644)
	if err != nil {
		return err
	}

	err = Chroot(opt.Mount)
	if err != nil {
		return err
	}
	defer ExitChroot()

	err = os.Chdir("/")
	if err != nil {
		return err
	}

	status, out, errr := utils.RunCMD("/setup.sh")
	if status != 0 {
		return fmt.Errorf("script /setup.sh failed with exitstatus: %d, stdout: %s, stderr: %s", status, out, errr)
	}

	return nil
}

func UmountHostPartitions(mnt string) {
	for _, p := range hostPartitions {
		Umount(filepath.Join(mnt, p))
	}
}

func Prepare4capture(opt PrepareOptions) error {
	switch dist := strings.ToLower(opt.Dist); dist {
	case "rhel", "centos":
		return prepare(opt)
	case "coreos":
		klog.Infof("No image preparation required for the coreos...")
		return nil
	default:
		return fmt.Errorf("not a supported distro: %s", dist)
	}
}
