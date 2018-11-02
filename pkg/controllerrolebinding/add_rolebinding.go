package controllerrolebinding

import "github.com/openshift/customer-admin-operator/pkg/controllerrolebinding/rolebinding"

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, rolebinding.Add)
}
