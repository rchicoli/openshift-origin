package node

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	kerrors "k8s.io/kubernetes/pkg/util/errors"
)

const (
	flagGracePeriod = "grace-period"
	flagDryRun      = "dry-run"
	flagForce       = "force"
)

type EvacuateOptions struct {
	Options *NodeOptions

	// Optional params
	DryRun      bool
	Force       bool
	GracePeriod int64
}

// NewEvacuateOptions creates a new EvacuateOptions with default values.
func NewEvacuateOptions(nodeOptions *NodeOptions) *EvacuateOptions {
	return &EvacuateOptions{
		Options:     nodeOptions,
		DryRun:      false,
		Force:       false,
		GracePeriod: 30,
	}
}

func (e *EvacuateOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&e.DryRun, flagDryRun, e.DryRun, "Show pods that will be migrated. Optional param for --evacuate")
	flags.BoolVar(&e.Force, flagForce, e.Force, "Delete pods not backed by replication controller. Optional param for --evacuate")
	flags.Int64Var(&e.GracePeriod, flagGracePeriod, e.GracePeriod, "Grace period (seconds) for pods being deleted. Optional param for --evacuate")

}

func (e *EvacuateOptions) Run() error {
	nodes, err := e.Options.GetNodes()
	if err != nil {
		return err
	}

	errList := []error{}
	for _, node := range nodes {
		err := e.RunEvacuate(node)
		if err != nil {
			// Don't bail out if one node fails
			errList = append(errList, err)
		}
	}
	return kerrors.NewAggregate(errList)
}

func (e *EvacuateOptions) RunEvacuate(node *kapi.Node) error {
	if e.DryRun {
		listpodsOp := ListPodsOptions{Options: e.Options}
		return listpodsOp.Run()
	}

	// We do *not* automatically mark the node unschedulable to perform evacuation.
	// Rationale: If we unschedule the node and later the operation is unsuccessful (stopped by user, network error, etc.),
	// we may not be able to recover in some cases to mark the node back to schedulable. To avoid these cases, we recommend
	// user to explicitly set the node to schedulable/unschedulable.
	if !node.Spec.Unschedulable {
		return fmt.Errorf("Node '%s' must be unschedulable to perform evacuation.\nYou can mark the node unschedulable with 'openshift admin manage-node %s --schedulable=false'", node.ObjectMeta.Name, node.ObjectMeta.Name)
	}

	labelSelector, err := labels.Parse(e.Options.PodSelector)
	if err != nil {
		return err
	}
	fieldSelector := fields.Set{GetPodHostFieldLabel(node.TypeMeta.APIVersion): node.ObjectMeta.Name}.AsSelector()

	// Filter all pods that satisfies pod label selector and belongs to the given node
	pods, err := e.Options.Kclient.Pods(kapi.NamespaceAll).List(kapi.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector})
	if err != nil {
		return err
	}
	rcs, err := e.Options.Kclient.ReplicationControllers(kapi.NamespaceAll).List(kapi.ListOptions{})
	if err != nil {
		return err
	}

	printerWithHeaders, printerNoHeaders, err := e.Options.GetPrintersByResource(unversioned.GroupVersionResource{Resource: "pod"})
	if err != nil {
		return err
	}

	errList := []error{}
	firstPod := true
	numPodsWithNoRC := 0
	deleteOptions := e.makeDeleteOptions()

	for _, pod := range pods.Items {
		foundrc := false
		for _, rc := range rcs.Items {
			selector := labels.SelectorFromSet(rc.Spec.Selector)
			if selector.Matches(labels.Set(pod.Labels)) {
				foundrc = true
				break
			}
		}

		if firstPod {
			fmt.Fprint(e.Options.ErrWriter, "\nMigrating these pods on node: ", node.ObjectMeta.Name, "\n\n")
			firstPod = false
			printerWithHeaders.PrintObj(&pod, e.Options.Writer)
		} else {
			printerNoHeaders.PrintObj(&pod, e.Options.Writer)
		}

		if foundrc || e.Force {
			if err := e.Options.Kclient.Pods(pod.Namespace).Delete(pod.Name, deleteOptions); err != nil {
				glog.Errorf("Unable to delete a pod: %+v, error: %v", pod, err)
				errList = append(errList, err)
				continue
			}
		} else { // Pods without replication controller and no --force option
			numPodsWithNoRC++
		}
	}
	if numPodsWithNoRC > 0 {
		err := fmt.Errorf(`Unable to evacuate some pods because they are not backed by replication controller.
Suggested options:
- You can list bare pods in json/yaml format using '--list-pods -o json|yaml'
- Force deletion of bare pods with --force option to --evacuate
- Optionally recreate these bare pods by massaging the json/yaml output from above list pods
`)
		errList = append(errList, err)
	}

	if len(errList) != 0 {
		return kerrors.NewAggregate(errList)
	}
	return nil
}

// makeDeleteOptions creates the delete options that will be used for pod evacuation.
func (e *EvacuateOptions) makeDeleteOptions() *kapi.DeleteOptions {
	return &kapi.DeleteOptions{GracePeriodSeconds: &e.GracePeriod}
}
