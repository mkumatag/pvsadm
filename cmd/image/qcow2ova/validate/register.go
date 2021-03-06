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

package validate

import (
	"github.com/ppc64le-cloud/pvsadm/cmd/image/qcow2ova/validate/diskspace"
	image_name "github.com/ppc64le-cloud/pvsadm/cmd/image/qcow2ova/validate/image-name"
	"github.com/ppc64le-cloud/pvsadm/cmd/image/qcow2ova/validate/platform"
	"github.com/ppc64le-cloud/pvsadm/cmd/image/qcow2ova/validate/tools"
	"github.com/ppc64le-cloud/pvsadm/cmd/image/qcow2ova/validate/user"
)

func init() {
	//TODO: Add Operating system check
	AddRule(&platform.Rule{})
	AddRule(&user.Rule{})
	AddRule(&image_name.Rule{})
	AddRule(&tools.Rule{})
	AddRule(&diskspace.Rule{})
}
