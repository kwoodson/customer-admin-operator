package common

import (
	"regexp"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (

	// RoleBindingNames holds the customer rolebindings
	RoleBindingNames = []string{
		"customer-admin",
		"customer-admin-project",
		"customer-view",
	}

	rxCustomerRoleBinding = regexp.MustCompile(`(?i)^` +
		`customer-(admin|view|admin-project)$`)

	rxRestrictedNamespace = regexp.MustCompile(`(?i)^` +
		`(kube|` +
		`openshift|` +
		`default|` +
		`kubernetes)-?`)
)

// RestrictedNamespace uses a regexp to match restricted namespaces
// namespaces include openshift, openshift-, kube, kube-, kubenetes, default
func RestrictedNamespace(name string) bool {
	return rxRestrictedNamespace.MatchString(name)
}

// CustomerRoleBinding uses a regexp to match rolebindings we manage
func CustomerRoleBinding(name string) bool {
	return rxCustomerRoleBinding.MatchString(name)
}

// verifyCustomerRoleBinding takes the existing rolebinding and the proposed
// and verifies that the subjects that are in the proposed exist in the
// existing rolebinding
func MissingSubjectsFromRoleBinding(incoming, existingRoleBinding *rbacv1.RoleBinding) []rbacv1.Subject {
	var missingSubjects []rbacv1.Subject
	if incoming.RoleRef.Name == existingRoleBinding.RoleRef.Name &&
		incoming.RoleRef.Kind == existingRoleBinding.RoleRef.Kind {
		for _, sub := range incoming.Subjects {
			if subjectInRolebindingSubjects(sub.Name, existingRoleBinding.Subjects) {
				continue
			}
			missingSubjects = append(missingSubjects, sub)
		}
	}
	return missingSubjects
}

func subjectInRolebindingSubjects(subject string, subjects []rbacv1.Subject) bool {
	for _, sub := range subjects {
		if subject == sub.Name {
			return true
		}
	}
	return false
}

// CustomerRoleBindingMap returns a rolebinding
func CustomerRoleBindingMap() func(string) rbacv1.RoleBinding {
	rbMap := map[string]rbacv1.RoleBinding{
		"customer-admin": rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "customer-admin",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: "customer-admins",
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "admin",
			},
		},
		"customer-admin-project": rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "customer-admin-project",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: "customer-admins",
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "customer-admin-project",
			},
		},
		"customer-view": rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "customer-view",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: "customer-readers",
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: "view",
			},
		},
	}

	return func(key string) rbacv1.RoleBinding {
		return rbMap[key]
	}
}
