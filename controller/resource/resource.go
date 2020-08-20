// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
package controller

import (
	"time"

	cf "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/common/fault/v3"
	hf "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	"github.com/golang/protobuf/ptypes/duration"

	"github.com/golang/protobuf/ptypes"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

var (
	ClusterName   = "sidecar_proxy_cluster"
	RouteName     = "local_route"
	ListenerName  = "listener_0"
	ListenerPort  = 10000
	UpstreamHost  = "127.0.0.1"
	UpstreamPort  = 80
	DelayDuration = 3
	Version       = 0
	Delay         = false
)

func SetDelay(duration int) {
	Delay = true
	if duration != 0 {
		DelayDuration = duration
	}
}

func CanselDelay() {
	Delay = false
}

func SetUpstream(host string, port int) {
	if host != "" {
		UpstreamHost = host
	}
	if port != 0 {
		UpstreamPort = port
	}
}

func SetListener(name string, port int) {
	if name != "" {
		ListenerName = name
	}
	if port != 0 {
		ListenerPort = port
	}
}

func IncVersion() {
	Version++
}

func makeCluster(clusterName string) *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_LOGICAL_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment:       makeEndpoint(clusterName),
	}
}

func makeEndpoint(clusterName string) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  UpstreamHost,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: uint32(UpstreamPort),
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

func makeRoute(routeName string, clusterName string) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "local_service",
			Domains: []string{"*"},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: clusterName,
						},
						HostRewriteSpecifier: &route.RouteAction_HostRewriteLiteral{
							HostRewriteLiteral: UpstreamHost,
						},
					},
				},
			}},
		}},
	}
}

func makeDelayHTTPListener(listenerName string, route string) *listener.Listener {
	// HTTP filter configuration
	fault := &hf.HTTPFault{
		Delay: &cf.FaultDelay{
			FaultDelaySecifier: &cf.FaultDelay_FixedDelay{
				FixedDelay: &duration.Duration{
					Seconds: int64(DelayDuration),
				},
			},
		},
	}
	pbft, err := ptypes.MarshalAny(fault)
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    makeConfigSource(),
				RouteConfigName: route,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Fault,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: pbft,
			},
		}, {
			Name: wellknown.Router,
		}},
	}

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(ListenerPort),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{
				{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: pbst,
					},
				}},
		}},
	}
}

func makeHTTPListener(listenerName string, route string) *listener.Listener {
	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    makeConfigSource(),
				RouteConfigName: route,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(ListenerPort),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{
				{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: pbst,
					},
				}},
		}},
	}
}

func makeConfigSource() *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       resource.DefaultAPIVersion,
			ApiType:                   core.ApiConfigSource_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
				},
			}},
		},
	}
	return source
}

func GenerateSnapshot() cache.Snapshot {
	if Delay {
		return cache.NewSnapshot(
			string(rune(Version)),
			[]types.Resource{}, // endpoints
			[]types.Resource{makeCluster(ClusterName)},
			[]types.Resource{makeRoute(RouteName, ClusterName)},
			[]types.Resource{makeDelayHTTPListener(ListenerName, RouteName)},
			[]types.Resource{}, // runtimes
		)
	} else {
		return cache.NewSnapshot(
			string(rune(Version)),
			[]types.Resource{}, // endpoints
			[]types.Resource{makeCluster(ClusterName)},
			[]types.Resource{makeRoute(RouteName, ClusterName)},
			[]types.Resource{makeHTTPListener(ListenerName, RouteName)},
			[]types.Resource{}, // runtimes
		)
	}

}
