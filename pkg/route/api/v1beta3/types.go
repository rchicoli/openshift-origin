package v1beta3

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1beta3"
	"k8s.io/kubernetes/pkg/util/intstr"
)

// Route encapsulates the inputs needed to connect an alias to endpoints.
type Route struct {
	unversioned.TypeMeta `json:",inline"`
	kapi.ObjectMeta      `json:"metadata,omitempty"`

	Spec   RouteSpec   `json:"spec"`
	Status RouteStatus `json:"status"`
}

// RouteList is a collection of Routes.
type RouteList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Route `json:"items"`
}

// RouteSpec describes the route the user wishes to exist.
type RouteSpec struct {
	// Ports are the ports that the user wishes to expose.
	//Ports []RoutePort `json:"ports,omitempty"`

	// Host is an alias/DNS that points to the service. Optional
	// Must follow DNS952 subdomain conventions.
	Host string `json:"host"`
	// Optional: Path that the router watches for, to route traffic for to the service
	Path string `json:"path,omitempty"`

	// An object the route points to. Only the Service kind is allowed, and it will
	// be defaulted to Service.
	To RouteTargetReference `json:"to"`

	// AlternateBackends is an extension of the 'to' field. If more than one service needs to be
	// pointed to, then use this field. Use the weight field in RouteTargetReference object
	// to specify relative preference
	AlternateBackends []RouteTargetReference `json:"alternateBackends,omitempty"`

	// If specified, the port to be used by the router. Most routers will use all
	// endpoints exposed by the service by default - set this value to instruct routers
	// which port to use.
	Port *RoutePort `json:"port,omitempty"`

	// TLS provides the ability to configure certificates and termination for the route
	TLS *TLSConfig `json:"tls,omitempty"`
}

// RouteTargetReference specifies the target that resolve into endpoints. Only the 'Service'
// kind is allowed. Use 'weight' field to emphasize one over others.
type RouteTargetReference struct {
	// The kind of target that the route is referring to. Currently, only 'Service' is allowed
	Kind string `json:"kind"`

	// Name of the service/target that is being referred to. e.g. name of the service
	Name string `json:"name"`

	// Weight as an integer between 1 and 256 that specifies the target's relative weight
	// against other target reference objects
	Weight *int32 `json:"weight"`
}

// RoutePort defines a port mapping from a router to an endpoint in the service endpoints.
type RoutePort struct {
	// The target port on pods selected by the service this route points to.
	// If this is a string, it will be looked up as a named port in the target
	// endpoints port list. Required
	TargetPort intstr.IntOrString `json:"targetPort"`
}

// RouteStatus provides relevant info about the status of a route, including which routers
// acknowledge it.
type RouteStatus struct {
	// Ingress describes the places where the route may be exposed. The list of
	// ingress points may contain duplicate Host or RouterName values. Routes
	// are considered live once they are `Ready`
	Ingress []RouteIngress `json:"ingress"`
}

// RouteIngress holds information about the places where a route is exposed
type RouteIngress struct {
	// Host is the host string under which the route is exposed; this value is required
	Host string `json:"host,omitempty"`
	// Name is a name chosen by the router to identify itself; this value is required
	RouterName string `json:"routerName,omitempty"`
	// Conditions is the state of the route, may be empty.
	Conditions []RouteIngressCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// RouteIngressConditionType is a valid value for RouteCondition
type RouteIngressConditionType string

// These are valid conditions of pod.
const (
	// RouteAdmitted means the route is able to service requests for the provided Host
	RouteAdmitted RouteIngressConditionType = "Admitted"
	// TODO: add other route condition types
)

// RouteIngressCondition contains details for the current condition of this pod.
// TODO: add LastTransitionTime, Reason, Message to match NodeCondition api.
type RouteIngressCondition struct {
	// Type is the type of the condition.
	// Currently only Ready.
	Type RouteIngressConditionType `json:"type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status kapi.ConditionStatus `json:"status"`
	// (brief) reason for the condition's last transition, and is usually a machine and human
	// readable constant
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
	// RFC 3339 date and time at which the object was acknowledged by the router.
	// This may be before the router exposes the route
	LastTransitionTime *unversioned.Time `json:"lastTransitionTime,omitempty"`
}

// RouterShard has information of a routing shard and is used to
// generate host names and routing table entries when a routing shard is
// allocated for a specific route.
// Caveat: This is WIP and will likely undergo modifications when sharding
//         support is added.
type RouterShard struct {
	// Shard name uniquely identifies a router shard in the "set" of
	// routers used for routing traffic to the services.
	ShardName string `json:"shardName"`

	// The DNS suffix for the shard ala: shard-1.v3.openshift.com
	DNSSuffix string `json:"dnsSuffix"`
}

// TLSConfig defines config used to secure a route and provide termination
type TLSConfig struct {
	// Termination indicates termination type.
	Termination TLSTerminationType `json:"termination"`

	// Certificate provides certificate contents
	Certificate string `json:"certificate,omitempty"`

	// Key provides key file contents
	Key string `json:"key,omitempty"`

	// CACertificate provides the cert authority certificate contents
	CACertificate string `json:"caCertificate,omitempty"`

	// DestinationCACertificate provides the contents of the ca certificate of the final destination.  When using reencrypt
	// termination this file should be provided in order to have routers use it for health checks on the secure connection
	DestinationCACertificate string `json:"destinationCACertificate,omitempty"`

	// InsecureEdgeTerminationPolicy indicates the desired behavior for
	// insecure connections to an edge-terminated route:
	//   disable, allow or redirect
	InsecureEdgeTerminationPolicy InsecureEdgeTerminationPolicyType `json:"insecureEdgeTerminationPolicy,omitempty"`
}

// TLSTerminationType dictates where the secure communication will stop
// TODO: Reconsider this type in v2
type TLSTerminationType string

// InsecureEdgeTerminationPolicyType dictates the behavior of insecure
// connections to an edge-terminated route.
type InsecureEdgeTerminationPolicyType string

const (
	// TLSTerminationEdge terminate encryption at the edge router.
	TLSTerminationEdge TLSTerminationType = "edge"
	// TLSTerminationPassthrough terminate encryption at the destination, the destination is responsible for decrypting traffic
	TLSTerminationPassthrough TLSTerminationType = "passthrough"
	// TLSTerminationReencrypt terminate encryption at the edge router and re-encrypt it with a new certificate supplied by the destination
	TLSTerminationReencrypt TLSTerminationType = "reencrypt"
)
