package prep

import (
	"bytes"
	"fmt"
	"text/template"
)

type CapiConfig struct{
	ContainerDURL string `json:"containerd_url"`
	ContainerDPauseImage string `json:"containerd_pause_image"`
	CrictlURL string `json:"crictl_url"`
	CNIURL string `json:"cni_url"`
}

func RenderCapi(dist, rhnuser, rhnpasswd, rootpasswd string) (string, error) {
	s := Setup{
		dist, rhnuser, rhnpasswd, rootpasswd,
	}
	var wr bytes.Buffer
	t := template.Must(template.New("setup").Parse(CapiSetupTemplate))
	err := t.Execute(&wr, s)
	if err != nil {
		return "", fmt.Errorf("error while rendoring the script template: %v", err)
	}
	return wr.String(), nil
}

var CapiSetupTemplate = `#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

set -x

mv /etc/resolv.conf /etc/resolv.conf.orig | true
echo "nameserver 9.9.9.9" | tee /etc/resolv.conf

#setup : add epel repo
yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm

yum update -y

yum install -y audit ca-certificates conntrack-tools chrony curl ebtables jq python3-pip socat sysstat yum-utils

cat <<EOF > /etc/modules-load.d/kubernetes.conf
overlay
br_netfilter
EOF

cat <<EOF >> /etc/sysctl.conf
net.bridge.bridge-nf-call-iptables=1
net.bridge.bridge-nf-call-ip6tables=1
net.ipv4.ip_forward=1
net.ipv6.conf.all.forwarding=1
net.ipv6.conf.all.disable_ipv6=0
net.ipv4.tcp_congestion_control=bbr
EOF

#systemctl disable conntrackd
# systemctl enable auditd
# ln -s /usr/lib/systemd/system/auditd.service /etc/systemd/system/multi-user.target.wants/auditd.service

cat <<EOF > /etc/audit/rules.d/containerd.rules
-w /var/lib/containerd/ -p rwxa -k containerd
-w /etc/containerd/ -p rwxa -k containerd
-w /etc/systemd/system/containerd.service -p rwxa -k containerd
-w /etc/systemd/system/containerd.service.d/ -p rwxa -k containerd
-w /run/containerd/ -p rwxa -k containerd
-w /usr/local/bin/containerd-shim -p rwxa -k containerd
-w /usr/local/bin/containerd-shim-runc-v1 -p rwxa -k containerd
-w /usr/local/bin/containerd-shim-runc-v2 -p rwxa -k containerd
-w /usr/local/sbin/runc -p rwxa -k containerd
-w /usr/local/bin/containerd -p rwxa -k containerd
EOF

# cloud-init

yum install http://people.redhat.com/~eterrell/cloud-init/cloud-init-19.4-11.el8_3.1.noarch.rpm -y
ln -s /usr/lib/systemd/system/cloud-init-local.service /etc/systemd/system/multi-user.target.wants/cloud-init-local.service
ln -s /usr/lib/systemd/system/cloud-init.service /etc/systemd/system/multi-user.target.wants/cloud-init.service
ln -s /usr/lib/systemd/system/cloud-config.service /etc/systemd/system/multi-user.target.wants/cloud-config.service
ln -s /usr/lib/systemd/system/cloud-final.service /etc/systemd/system/multi-user.target.wants/cloud-final.service
rm -rf /etc/systemd/system/multi-user.target.wants/firewalld.service
rpm -vih --nodeps http://public.dhe.ibm.com/software/server/POWER/Linux/yum/download/ibm-power-repo-latest.noarch.rpm
sed -i 's/^more \/opt\/ibm\/lop\/notice/#more \/opt\/ibm\/lop\/notice/g' /opt/ibm/lop/configure
echo 'y' | /opt/ibm/lop/configure
# Disable the AT repository due to slowness in nature
yum-config-manager --disable Advance_Toolchain
yum install  powerpc-utils librtas DynamicRM  devices.chrp.base.ServiceRM rsct.opt.storagerm rsct.core rsct.basic rsct.core src -y

mkdir -p /etc/systemd/system/cloud-config.service.d
mkdir -p /etc/systemd/system/cloud-final.service.d

cat <<EOF > /etc/systemd/system/cloud-config.service.d/boot-order.conf
[Unit]
After=containerd.service
Wants=containerd.service
EOF

cat <<EOF > /etc/systemd/system/cloud-final.service.d/boot-order.conf
[Unit]
After=containerd.service
Wants=containerd.service
EOF

yum install -y libseccomp

curl -L https://github.com/mkumatag/containerd/releases/download/v1.4.3/cri-containerd-cni-1.4.3.linux-ppc64le.tar.gz -o /tmp/containerd.tar.gz

/bin/gtar --extract -C / -z --show-transformed-names --no-overwrite-dir -f /tmp/containerd.tar.gz

rm -rf /opt/cni
rm -rf /etc/cni

mkdir -p /etc/systemd/system/containerd.service.d

cat <<EOF > /etc/systemd/system/containerd.service.d/boot-order.conf
[Unit]
After=cloud-init.service
Wants=cloud-init.service
EOF

cat <<EOF > /etc/systemd/system/containerd.service.d/memory-pressure.conf
[Service]
OOMScoreAdjust=-999
EOF

cat <<EOF > /etc/systemd/system/containerd.service.d/max-tasks.conf
[Service]
# Do not limit the number of tasks that can be spawned by containerd
TasksMax=infinity
EOF

mkdir -p /etc/containerd/

cat <<EOF > /etc/containerd/config.toml
version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "k8s.gcr.io/pause:3.2"
EOF

cat <<EOF > /etc/crictl.yaml
runtime-endpoint: unix:///var/run/containerd/containerd.sock
EOF

# systemctl enable containerd
ln -s /usr/lib/systemd/system/containerd.service /etc/systemd/system/multi-user.target.wants/containerd.service

rm -rf /tmp/containerd.tar.gz

cat <<EOF | tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-\$basearch
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
exclude=kubelet kubeadm kubectl
EOF

yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes

# systemctl enable kubelet
ln -s /usr/lib/systemd/system/kubelet.service /etc/systemd/system/multi-user.target.wants/kubelet.service

yum -y clean all

rm -rf /etc/sysconfig/network-scripts/ifcfg-eth0
echo {{ .RootPasswd }} | passwd root --stdin

# Remove the ibm repositories used for the rsct installation
rpm -e ibm-power-repo-*.noarch

mv /etc/resolv.conf.orig /etc/resolv.conf | true
touch /.autorelabel
`