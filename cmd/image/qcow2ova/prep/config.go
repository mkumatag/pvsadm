package prep

//yum_repos:
//- name: open-power-unicamp.repo
//  content: |
//   [Open-Power]
//   name=Unicamp OpenPower Lab - $basearch
//   baseurl=https://oplab9.parqtec.unicamp.br/pub/repository/rpm/
//   enabled=1
//   gpgcheck=0
//   repo_gpgcheck=1
//   gpgkey=https://oplab9.parqtec.unicamp.br/pub/key/openpower-gpgkey-public.asc

var DefaultAdditionalConfig = `
nameservers:
  - 9.9.9.9
  - 8.8.8.8
yum_repos:
- name: open-power-unicamp.repo
  content: |
    [Open-Power]
    name=Unicamp OpenPower Lab - \$basearch
    baseurl=https://oplab9.parqtec.unicamp.br/pub/repository/rpm/
    enabled=1
    gpgcheck=0
    repo_gpgcheck=1
    gpgkey=https://oplab9.parqtec.unicamp.br/pub/key/openpower-gpgkey-public.asc
yum_install:
  - http://people.redhat.com/~eterrell/cloud-init/cloud-init-19.4-11.el8_3.1.noarch.rpm
`

type Setup struct {
	// Distribution type e.g: rhel, coreos
	Dist string

	// User name for RHN, applied only for RHEL distro
	RHNUser string

	// Password for RHN, applied only for RHEL distro
	RHNPassword string

	// Root password
	RootPasswd string
}

type AdditionalSetup struct {
	NameServers []string  `yaml:"nameservers"`
	YumRepos    []YumRepo `yaml:"yum_repos,flow"`
	// Invokes rpm -i command
	RPMInstall []string `yaml:"rpm_install"`
	// Invokes yum install -y command
	YumInstall []string `yaml:"yum_install"`
}

type YumRepo struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`
}
