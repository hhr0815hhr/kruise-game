package gss

import (
	"context"
	"encoding/json"
	cerr "errors"
	gamekruiseiov1alpha1 "github.com/openkruise/kruise-game/apis/v1alpha1"
	"github.com/openkruise/kruise-game/cloudprovider"
	"github.com/openkruise/kruise-game/cloudprovider/errors"
	"github.com/openkruise/kruise-game/cloudprovider/utils"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// 此插件依赖SR-IOV分配VF来实现ip直连功能,使用前需检查当前CNI是否支持，并安装该CNI对应的SR-IOV网络插件
type EipPlugin struct {
}

type EipDefinition struct {
	Name         string   `json:"name,omitempty"`
	Namespace    string   `json:"namespace,omitempty"`
	Interface    string   `json:"interface,omitempty"`
	DefaultRoute []string `json:"default-route,omitempty"`
}

type EipState struct {
	Name      string
	Interface string
	Ips       []string
	Mac       string
}

const (
	EIPNetwork                    = "Gss-EIP"
	AliasSEIP                     = "EIP-Network"
	CniNetworkAttachConfigName    = "CniAttach"
	CniNamespaceConfigName        = "CniNamespace"
	CniInterfaceConfigName        = "CniInterface"
	CniGateWayConfigName          = "CniGateway"
	FixedEIPConfigName            = "Fixed"
	EnableEIPAnnotationConfigName = "Enable"

	EnableEIPAnnotationKey = "gss-eip-enable"
	CniNetworkStateKey     = "k8s.v1.cni.cncf.io/network-status"
	CniNetworkAttachKey    = "k8s.v1.cni.cncf.io/networks"
	FixedEIPKey            = "gss-fixed"
)

func (e EipPlugin) Name() string {
	return EIPNetwork
}

func (e EipPlugin) Alias() string {
	return AliasSEIP
}

func (e EipPlugin) Init(client client.Client, options cloudprovider.CloudProviderOptions, ctx context.Context) error {
	return nil
}

func (e EipPlugin) OnPodAdded(client client.Client, pod *corev1.Pod, ctx context.Context) (*corev1.Pod, errors.PluginError) {
	networkManager := utils.NewNetworkManager(pod, client)
	conf := networkManager.GetNetworkConfig()
	//parse network configuration
	var df EipDefinition
	for _, c := range conf {
		switch c.Name {
		case CniNetworkAttachConfigName:
			df.Name = c.Value
		case FixedEIPConfigName:
			pod.Annotations[FixedEIPKey] = c.Value
		case EnableEIPAnnotationConfigName:
			pod.Annotations[EnableEIPAnnotationKey] = c.Value
		case CniNamespaceConfigName:
			df.Namespace = c.Value
		case CniInterfaceConfigName:
			df.Interface = c.Value
		case CniGateWayConfigName:
			df.DefaultRoute = strings.Split(c.Value, ",")
		}
	}
	dfJson, err := json.Marshal([]EipDefinition{df})
	if err != nil {
		return pod, errors.ToPluginError(cerr.New("gss-eip params is invalid"), errors.InternalError)
	}
	pod.Annotations[CniNetworkAttachKey] = string(dfJson)
	return pod, nil
}

func (e EipPlugin) OnPodUpdated(client client.Client, pod *corev1.Pod, ctx context.Context) (*corev1.Pod, errors.PluginError) {
	networkManager := utils.NewNetworkManager(pod, client)

	networkStatus, _ := networkManager.GetNetworkStatus()
	if networkStatus == nil {
		pod, err := networkManager.UpdateNetworkStatus(gamekruiseiov1alpha1.NetworkStatus{
			CurrentNetworkState: gamekruiseiov1alpha1.NetworkWaiting,
		}, pod)
		return pod, errors.ToPluginError(err, errors.InternalError)
	}

	if enable, ok := pod.Annotations[EnableEIPAnnotationKey]; !ok || (ok && enable != "true") {
		return pod, errors.ToPluginError(cerr.New("gss-eip plugin is not enabled"), errors.InternalError)
	}
	if _, ok := pod.Annotations[CniNetworkStateKey]; !ok {
		return pod, nil
	}

	eip, err := _getEipFromState(pod.Annotations[CniNetworkStateKey])
	if err != nil {
		return pod, errors.ToPluginError(cerr.New("gss-eip network status is invalid"), errors.InternalError)
	}
	networkStatus.ExternalAddresses = []gamekruiseiov1alpha1.NetworkAddress{
		{
			IP: eip,
		},
	}
	networkStatus.InternalAddresses = []gamekruiseiov1alpha1.NetworkAddress{
		{
			IP: pod.Status.PodIP,
		},
	}
	networkStatus.CurrentNetworkState = gamekruiseiov1alpha1.NetworkReady

	pod, err = networkManager.UpdateNetworkStatus(*networkStatus, pod)
	return pod, errors.ToPluginError(err, errors.InternalError)
}

func (e EipPlugin) OnPodDeleted(client client.Client, pod *corev1.Pod, ctx context.Context) errors.PluginError {
	return nil
}

func init() {
	gssProvider.registerPlugin(&EipPlugin{})
}

func _getEipFromState(state string) (string, error) {
	if state == "" {
		return "", cerr.New("eip network state is empty")
	}
	var eipState []EipState
	err := json.Unmarshal([]byte(state), &eipState)
	if err != nil {
		return "", cerr.New("eip network state is invalid")
	}
	if len(eipState) == 0 || len(eipState[0].Ips) == 0 {
		return "", cerr.New("eip is empty")
	}
	return eipState[0].Ips[0], nil
}
