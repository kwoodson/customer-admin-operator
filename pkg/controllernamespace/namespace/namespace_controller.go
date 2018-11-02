package namespace

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/openshift/customer-admin-operator/pkg/common"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CustomerAdminReconciler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNamespace{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("customeradminreconciler-namespace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// TODO: figure out how to filter on namespaces that are !prefixed by a certain string
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNamespace{}

// ReconcileNamespace reconciles a CustomerAdminReconciler object
type ReconcileNamespace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile receives a namespace event and reconciles rolebindings for
// customer roles
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNamespace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if !common.RestrictedNamespace(request.Name) {
		// move to appropriate place
		// If we are namespace we need to reconcile 3 rolebindings:
		// - customer-admin
		// - customer-project-admin
		// - customer-view
		log.Printf("Reconciling Namespace %s/%s\n", request.Namespace, request.Name)

		// check for existence of namespace
		ns := &corev1.Namespace{}
		err := r.client.Get(context.TODO(), request.NamespacedName, ns)
		if err != nil {
			if kerrors.IsNotFound(err) {
				log.Printf("error: %v", err)
				return reconcile.Result{}, nil
			}
		}
		// do not reconcile. namespace is being deleted
		if ns.Status.Phase == corev1.NamespaceTerminating {
			log.Printf("namespace is being deleted %v", request.Namespace)
			return reconcile.Result{}, nil
		}

		nn := types.NamespacedName{
			Namespace: request.Name,
		}
		for _, rbname := range common.RoleBindingNames {
			nrb := common.CustomerRoleBindingMap()(rbname)
			nrb.Namespace = request.Name
			nn.Name = rbname
			rb := &rbacv1.RoleBinding{}
			err := r.client.Get(context.TODO(), nn, rb)
			if err != nil {
				if kerrors.IsNotFound(err) {
					// need to create rolebinding
					log.Printf("Creating rolebinding: %#v", nrb)
					err = r.client.Create(context.TODO(), &nrb)
					if err != nil {
						log.Printf("Error Creating: %v\n", err)
						return reconcile.Result{}, err
					}
					continue
				}

				return reconcile.Result{}, err
			}
			// check that our rolebinding is correct by seeing if the roleref
			// and subjects are in desired state
			missingSubjects := common.MissingSubjectsFromRoleBinding(&nrb, rb)
			if len(missingSubjects) == 0 {
				log.Printf("Rolebinding %v looks ok!\n", rbname)
				continue // rolebinding looks ok
			}

			// append missing subjects
			rb.Subjects = append(rb.Subjects, missingSubjects...)
			log.Printf("Updating rolebinging[%v]", rbname)
			err = r.client.Update(context.TODO(), rb)
			if err != nil {
				log.Printf("Error updating rolebinding: %v\n", err)
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}
