/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/provider-template/internal/controller/application"
	"github.com/crossplane/provider-template/internal/controller/client"
	"github.com/crossplane/provider-template/internal/controller/config"
	"github.com/crossplane/provider-template/internal/controller/emailtemplate"
	"github.com/crossplane/provider-template/internal/controller/group"
	mdmapplication "github.com/crossplane/provider-template/internal/controller/mdm/application"
	mdmauthenticationmethod "github.com/crossplane/provider-template/internal/controller/mdm/authenticationmethod"
	mdmdevicegroup "github.com/crossplane/provider-template/internal/controller/mdm/devicegroup"
	mdmdevicetype "github.com/crossplane/provider-template/internal/controller/mdm/devicetype"
	mdmproposition "github.com/crossplane/provider-template/internal/controller/mdm/proposition"
	mdmstandardservice "github.com/crossplane/provider-template/internal/controller/mdm/standardservice"
	"github.com/crossplane/provider-template/internal/controller/organization"
	"github.com/crossplane/provider-template/internal/controller/passwordpolicy"
	"github.com/crossplane/provider-template/internal/controller/proposition"
	provisioningorgconfiguration "github.com/crossplane/provider-template/internal/controller/provisioning/orgconfiguration"
	"github.com/crossplane/provider-template/internal/controller/role"
	"github.com/crossplane/provider-template/internal/controller/service"
	"github.com/crossplane/provider-template/internal/controller/user"
)

// SetupGated creates all DIP controllers with safe-start support and adds them to
// the supplied manager.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		// IAM resources
		organization.Setup,
		group.Setup,
		role.Setup,
		proposition.Setup,
		application.Setup,
		client.Setup,
		service.Setup,
		user.Setup,
		emailtemplate.Setup,
		passwordpolicy.Setup,
		// MDM resources
		mdmproposition.Setup,
		mdmapplication.Setup,
		mdmstandardservice.Setup,
		mdmdevicegroup.Setup,
		mdmdevicetype.Setup,
		mdmauthenticationmethod.Setup,
		// Provisioning resources
		provisioningorgconfiguration.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
