package controller

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/aquasecurity/starboard/pkg/kube"

	"github.com/aquasecurity/starboard-octant-plugin/pkg/plugin/view/configaudit"
	"github.com/aquasecurity/starboard-octant-plugin/pkg/plugin/view/kubebench"
	"github.com/aquasecurity/starboard-octant-plugin/pkg/plugin/view/vulnerabilities"

	"github.com/aquasecurity/starboard-octant-plugin/pkg/plugin/model"

	"github.com/vmware-tanzu/octant/pkg/plugin"
	"github.com/vmware-tanzu/octant/pkg/plugin/service"
	"github.com/vmware-tanzu/octant/pkg/view/component"
	"k8s.io/apimachinery/pkg/api/meta"
)

// ResourceTabPrinter is called when Octant wants to add new tab for the underlying resource.
func ResourceTabPrinter(request *service.PrintRequest) (tab plugin.TabResponse, err error) {
	if request.Object == nil {
		err = errors.New("request object is nil")
		return
	}

	workload, err := getWorkloadFromObject(request.Object)
	if err != nil {
		return
	}

	switch workload.Kind {
	case kube.KindPod,
		kube.KindDeployment,
		kube.KindDaemonSet,
		kube.KindStatefulSet,
		kube.KindReplicaSet,
		kube.KindReplicationController,
		kube.KindCronJob,
		kube.KindJob:
		return vulnerabilitiesTabPrinter(request, workload)
	case kube.KindNode:
		return cisKubernetesBenchmarksTabPrinter(request, workload.Name)
	default:
		err = fmt.Errorf("unrecognized workload kind: %s", workload.Kind)
		return
	}

}

func vulnerabilitiesTabPrinter(request *service.PrintRequest, workload kube.Object) (tabResponse plugin.TabResponse, err error) {
	repository := model.NewRepository(request.DashboardClient)
	reports, err := repository.GetVulnerabilitiesForWorkload(request.Context(), workload)
	if err != nil {
		return
	}

	tab := component.NewTabWithContents(vulnerabilities.NewReport(workload, reports))
	tabResponse = plugin.TabResponse{Tab: tab}

	return
}

func cisKubernetesBenchmarksTabPrinter(request *service.PrintRequest, node string) (tabResponse plugin.TabResponse, err error) {
	repository := model.NewRepository(request.DashboardClient)
	report, err := repository.GetCISKubeBenchReport(request.Context(), node)
	if err != nil {
		return
	}

	tab := component.NewTabWithContents(kubebench.NewReport(report))
	tabResponse = plugin.TabResponse{Tab: tab}
	return
}

// ResourcePrinter is called when Octant wants to print the details of the underlying resource.
func ResourcePrinter(request *service.PrintRequest) (response plugin.PrintResponse, err error) {
	if request.Object == nil {
		err = errors.New("object is nil")
		return
	}

	repository := model.NewRepository(request.DashboardClient)

	workload, err := getWorkloadFromObject(request.Object)
	if err != nil {
		return
	}

	summary, err := repository.GetVulnerabilitiesSummary(request.Context(), workload)
	if err != nil {
		return
	}

	configAudit, err := repository.GetConfigAudit(request.Context(), workload)
	if err != nil {
		return
	}

	response = plugin.PrintResponse{
		Status: vulnerabilities.NewSummarySections(summary),
		Items: []component.FlexLayoutItem{
			{
				Width: component.WidthFull,
				View:  configaudit.NewReport(configAudit),
			},
		},
	}
	return
}

func getWorkloadFromObject(o runtime.Object) (workload kube.Object, err error) {
	accessor := meta.NewAccessor()

	kind, err := accessor.Kind(o)
	if err != nil {
		return
	}

	name, err := accessor.Name(o)
	if err != nil {
		return
	}

	namespace, err := accessor.Namespace(o)
	if err != nil {
		return
	}

	workload = kube.Object{
		Kind:      kube.Kind(kind),
		Name:      name,
		Namespace: namespace,
	}
	return
}
