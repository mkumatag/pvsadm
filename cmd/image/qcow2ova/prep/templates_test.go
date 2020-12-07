package prep

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		args    PrepareOptions
		want    string
		wantErr bool
	}{
		{
			"rhel image",
			PrepareOptions{
				Setup: Setup{
					Dist: "rhel",
				},
			},
			"subscription-manager",
			false,
		},
		{
			"centos image",
			PrepareOptions{
				Setup: Setup{
					Dist:       "centos",
					RootPasswd: "some-password",
				},
			},
			"some-password",
			false,
		},
		{
			"nameservers",
			PrepareOptions{
				Setup: Setup{
					Dist:       "centos",
					RootPasswd: "some-password",
				},
				AdditionalSetup: AdditionalSetup{
					NameServers: []string{"1.2.3.4", "2.3.4.5"},
				},
			},
			"nameserver 1.2.3.4",
			false,
		},
		{
			"User defined yum repository",
			PrepareOptions{
				Setup: Setup{
					Dist:       "centos",
					RootPasswd: "some-password",
				},
				AdditionalSetup: AdditionalSetup{
					NameServers: []string{"1.2.3.4", "2.3.4.5"},
					YumRepos: []YumRepo{
						{
							Name:    "openpower.repo",
							Content: "   [Open-Power]\n   name=Unicamp OpenPower Lab - $basearch\n   baseurl=https://oplab9.parqtec.unicamp.br/pub/repository/rpm/\n   enabled=1\n   gpgcheck=0\n   repo_gpgcheck=1\n   gpgkey=https://oplab9.parqtec.unicamp.br/pub/key/openpower-gpgkey-public.asc",
						},
					},
				},
			},
			"/etc/yum.repos.d/openpower.repo",
			false,
		},
		{
			"Install package via rpm",
			PrepareOptions{
				Setup: Setup{
					Dist:       "centos",
					RootPasswd: "some-password",
				},
				AdditionalSetup: AdditionalSetup{
					RPMInstall: []string{"some-package"},
				},
			},
			"rpm -i some-package",
			false,
		},
		{
			"Install package via yum",
			PrepareOptions{
				Setup: Setup{
					Dist:       "centos",
					RootPasswd: "some-password",
				},
				AdditionalSetup: AdditionalSetup{
					YumInstall: []string{"some-package"},
				},
			},
			"yum install -y some-package",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("Render() %s does not contain the %s", got, tt.want)
			}
		})
	}
}
