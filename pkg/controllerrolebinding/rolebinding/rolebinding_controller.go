package rolebinding

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/openshift/customer-admin-operator/pkg/common"
)

// Add creates a new CustomerAdminReconciler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRolebinding{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("customeradminreconciler-rolebinding-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// TODO: figure out how to only return specific rolebindings
	err = c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRolebinding{}

// ReconcileRolebinding reconciles a CustomerAdminReconciler object
type ReconcileRolebinding struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile receives events for rolebindings
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRolebinding) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if !common.RestrictedNamespace(request.Name) {
		if common.CustomerRoleBinding(request.Name) {
			log.Printf("Reconcile rolebinding %v", request.Name)

			// check for existence of namespace
			ns := &corev1.Namespace{}
			nn := types.NamespacedName{
				Name: request.Namespace,
			}
			err := r.client.Get(context.TODO(), nn, ns)
			if err != nil {
				if kerrors.IsNotFound(err) {
					log.Printf("Error: %v", err)
					return reconcile.Result{}, nil
				}
			}
			// do not reconcile. namespace is being deleted
			if ns.Status.Phase == corev1.NamespaceTerminating {
				log.Printf("namespace is being deleted %v", request.Namespace)
				return reconcile.Result{}, nil
			}

			// We have the nn so fetch the rolebinding
			nrb := common.CustomerRoleBindingMap()(request.Name)
			nrb.Namespace = request.Namespace
			rb := &rbacv1.RoleBinding{}
			//check for rolebinding existence
			err = r.client.Get(context.TODO(), request.NamespacedName, rb)
			if err != nil {
				if kerrors.IsNotFound(err) {
					// we didn't find our rolebinding, create it!
					log.Printf("Creating rolebinding: %#v", nrb)
					err = r.client.Create(context.TODO(), &nrb)
					if err != nil {
						log.Printf("Error Creating: %v\n", err)
						return reconcile.Result{}, err
					}
				}
			}
			// check that our rolebinding is correct by seeing if the roleref
			// and subjects are in desired state
			missingSubjects := common.MissingSubjectsFromRoleBinding(&nrb, rb)
			if len(missingSubjects) == 0 {
				log.Printf("Rolebinding %v looks ok!", request.Name)
				return reconcile.Result{}, nil
			}

			// append missing subjects
			rb.Subjects = append(rb.Subjects, missingSubjects...)
			log.Printf("Updating rolebinging[%v]\n", request.Name)
			err = r.client.Update(context.TODO(), rb)
			if err != nil {
				log.Printf("Error updating rolebinding: %v\n", err)
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}
