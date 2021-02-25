package kt

import (
	"strconv"
)

type Compression string

const (
	CompressionNone    Compression = "none"
	CompressionGzip                = "gzip"
	CompressionSnappy              = "snappy"
	CompressionDeflate             = "deflate"
	CompressionNull                = "null"

	KENTIK_EVENT_TYPE  = "KFlow"
	KENTIK_EVENT_SNMP  = "KSnmp"
	KENTIK_EVENT_TRACE = "KTrace"
	KENTIK_EVENT_SYNTH = "KSynth"

	KENTIK_EVENT_SNMP_DEV_METRIC = "KSnmpDeviceMetric"
	KENTIK_EVENT_SNMP_INT_METRIC = "KSnmpInterfaceMetric"
	KENTIK_EVENT_SNMP_METADATA   = "KSnmpInterfaceMetadata"
	KENTIK_EVENT_SNMP_TRAP       = "KSnmpTrap"
)

type IntId uint64

type Cid IntId              // company id
func (id Cid) Itoa() string { return strconv.Itoa(int(id)) }

// DeviceID denotes id from mn_device table
type DeviceID IntId

// Itoa returns string formated device id
func (id DeviceID) Itoa() string {
	return strconv.FormatInt(int64(id), 10)
}

// IfaceID denotes interfce id
// note this is not mn_interface.id but snmp_id, {input,output}_port in flow
type IfaceID IntId

// Itoa returns string repr of interface id
func (id IfaceID) Itoa() string { return strconv.Itoa(int(id)) }

type JCHF struct {
	Timestamp               int64             `json:"timestamp"`
	DstAs                   uint32            `json:"dst_as"`
	DstGeo                  string            `json:"dst_geo"`
	HeaderLen               uint32            `json:"header_len"`
	InBytes                 uint64            `json:"in_bytes"`
	InPkts                  uint64            `json:"in_pkts"`
	InputPort               IfaceID           `json:"input_port"`
	IpSize                  uint32            `json:"ip_size"`
	DstAddr                 string            `json:"dst_addr"`
	SrcAddr                 string            `json:"src_addr"`
	L4DstPort               uint32            `json:"l4_dst_port"`
	L4SrcPort               uint32            `json:"l4_src_port"`
	OutputPort              IfaceID           `json:"output_port"`
	Protocol                uint32            `json:"protocol"`
	SampledPacketSize       uint32            `json:"sampled_packet_size"`
	SrcAs                   uint32            `json:"src_as"`
	SrcGeo                  string            `json:"src_geo"`
	TcpFlags                uint32            `json:"tcp_flags"`
	Tos                     uint32            `json:"tos"`
	VlanIn                  uint32            `json:"vlan_in"`
	VlanOut                 uint32            `json:"vlan_out"`
	NextHop                 string            `json:"next_hop"`
	MplsType                uint32            `json:"mpls_type"`
	OutBytes                uint64            `json:"out_bytes"`
	OutPkts                 uint64            `json:"out_pkts"`
	TcpRetransmit           uint32            `json:"tcp_rx"`
	SrcFlowTags             string            `json:"src_flow_tags"`
	DstFlowTags             string            `json:"dst_flow_tags"`
	SampleRate              uint32            `json:"sample_rate"`
	DeviceId                DeviceID          `json:"device_id"`
	DeviceName              string            `json:"device_name"`
	CompanyId               Cid               `json:"company_id"`
	DstBgpAsPath            string            `json:"dst_bgp_as_path"`
	DstBgpCommunity         string            `json:"dst_bgp_comm"`
	SrcBgpAsPath            string            `json:"src_bpg_as_path"`
	SrcBgpCommunity         string            `json:"src_bgp_comm"`
	SrcNextHopAs            uint32            `json:"src_nexthop_as"`
	DstNextHopAs            uint32            `json:"dst_nexthop_as"`
	SrcGeoRegion            string            `json:"src_geo_region"`
	DstGeoRegion            string            `json:"dst_geo_region"`
	SrcGeoCity              string            `json:"src_geo_city"`
	DstGeoCity              string            `json:"dst_geo_city"`
	DstNextHop              string            `json:"dst_nexthop"`
	SrcNextHop              string            `json:"src_nexthop"`
	SrcRoutePrefix          uint32            `json:"src_route_prefix"`
	DstRoutePrefix          uint32            `json:"dst_route_prefix"`
	SrcSecondAsn            uint32            `json:"src_second_asn"`
	DstSecondAsn            uint32            `json:"dst_second_asn"`
	SrcThirdAsn             uint32            `json:"src_third_asn"`
	DstThirdAsn             uint32            `json:"dst_third_asn"`
	SrcEthMac               string            `json:"src_eth_mac"`
	DstEthMac               string            `json:"dst_eth_mac"`
	InputIntDesc            string            `json:"input_int_desc"`
	OutputIntDesc           string            `json:"output_int_desc"`
	InputIntAlias           string            `json:"input_int_alias"`
	OutputIntAlias          string            `json:"output_int_alias"`
	InputInterfaceCapacity  int64             `json:"input_int_capacity"`
	OutputInterfaceCapacity int64             `json:"output_int_capacity"`
	InputInterfaceIP        string            `json:"input_int_ip"`
	OutputInterfaceIP       string            `json:"output_int_ip"`
	CustomStr               map[string]string `json:"custom_str,omitempty"`
	CustomInt               map[string]int32  `json:"custom_int,omitempty"`
	CustomBigInt            map[string]int64  `json:"custom_bigint,omitempty"`
	EventType               string            `json:"eventType"`
	avroSet                 map[string]interface{}
	hasSetAvro              bool
}

func NewJCHF() *JCHF {
	return &JCHF{avroSet: map[string]interface{}{}, hasSetAvro: false}
}

func (j *JCHF) Reset() {
	j.hasSetAvro = false
	//j.SetMap() @TODO -- don't think this is needed?
}

func (j *JCHF) Flatten() map[string]interface{} {
	mapr := j.ToMap()
	for k, v := range mapr {
		switch mv := v.(type) {
		case map[string]string:
			for ki, vi := range mv {
				mapr[ki] = vi
			}
			delete(mapr, k)
		case map[string]int32:
			for ki, vi := range mv {
				mapr[ki] = vi
			}
			delete(mapr, k)
		case map[string]int64:
			for ki, vi := range mv {
				mapr[ki] = vi
			}
			delete(mapr, k)
		case string:
			if mv == "<nil>" {
				mapr[k] = ""
			}
		default:
			// noop
		}
	}
	return mapr
}

func (j *JCHF) ToMap() map[string]interface{} {
	if j.hasSetAvro { //cache this also.
		return j.avroSet
	}

	j.avroSet["timestamp"] = int64(j.Timestamp)
	j.avroSet["dst_as"] = int64(j.DstAs)
	j.avroSet["dst_geo"] = j.DstGeo
	j.avroSet["header_len"] = int64(j.HeaderLen)
	j.avroSet["in_bytes"] = int64(j.InBytes)
	j.avroSet["in_pkts"] = int64(j.InPkts)
	j.avroSet["input_port"] = int64(j.InputPort)
	j.avroSet["dst_addr"] = j.DstAddr
	j.avroSet["src_addr"] = j.SrcAddr
	j.avroSet["l4_dst_port"] = int64(j.L4DstPort)
	j.avroSet["l4_src_port"] = int64(j.L4SrcPort)
	j.avroSet["output_port"] = int64(j.OutputPort)
	j.avroSet["protocol"] = int64(j.Protocol)
	j.avroSet["sampled_packet_size"] = int64(j.SampledPacketSize)
	j.avroSet["src_as"] = int64(j.SrcAs)
	j.avroSet["src_geo"] = j.SrcGeo
	j.avroSet["tcp_flags"] = int64(j.TcpFlags)
	j.avroSet["tos"] = int64(j.Tos)
	j.avroSet["vlan_in"] = int64(j.VlanIn)
	j.avroSet["vlan_out"] = int64(j.VlanOut)
	j.avroSet["out_bytes"] = int64(j.OutBytes)
	j.avroSet["out_pkts"] = int64(j.OutPkts)
	j.avroSet["tcp_rx"] = int64(j.TcpRetransmit)
	j.avroSet["src_flow_tags"] = j.SrcFlowTags
	j.avroSet["dst_flow_tags"] = j.DstFlowTags
	j.avroSet["sample_rate"] = int64(j.SampleRate)
	j.avroSet["device_id"] = int64(j.DeviceId)
	j.avroSet["device_name"] = j.DeviceName
	j.avroSet["company_id"] = int64(j.CompanyId)
	j.avroSet["dst_bgp_as_path"] = j.DstBgpAsPath
	j.avroSet["dst_bgp_comm"] = j.DstBgpCommunity
	j.avroSet["src_bpg_as_path"] = j.SrcBgpAsPath
	j.avroSet["src_bgp_comm"] = j.SrcBgpCommunity
	j.avroSet["src_nexthop_as"] = int64(j.SrcNextHopAs)
	j.avroSet["dst_nexthop_as"] = int64(j.DstNextHopAs)
	j.avroSet["src_geo_region"] = j.SrcGeoRegion
	j.avroSet["dst_geo_region"] = j.DstGeoRegion
	j.avroSet["src_geo_city"] = j.SrcGeoCity
	j.avroSet["dst_geo_city"] = j.DstGeoCity
	j.avroSet["dst_nexthop"] = j.DstNextHop
	j.avroSet["src_nexthop"] = j.SrcNextHop
	j.avroSet["src_route_prefix"] = int64(j.SrcRoutePrefix)
	j.avroSet["dst_route_prefix"] = int64(j.DstRoutePrefix)
	j.avroSet["src_second_asn"] = int64(j.SrcSecondAsn)
	j.avroSet["dst_second_asn"] = int64(j.DstSecondAsn)
	j.avroSet["src_third_asn"] = int64(j.SrcThirdAsn)
	j.avroSet["dst_third_asn"] = int64(j.DstThirdAsn)
	j.avroSet["src_eth_mac"] = j.SrcEthMac
	j.avroSet["dst_eth_mac"] = j.DstEthMac
	j.avroSet["input_int_desc"] = j.InputIntDesc
	j.avroSet["output_int_desc"] = j.OutputIntDesc
	j.avroSet["input_int_alias"] = j.InputIntAlias
	j.avroSet["output_int_alias"] = j.OutputIntAlias
	j.avroSet["input_interface_capacity"] = j.InputInterfaceCapacity
	j.avroSet["output_interface_capacity"] = j.OutputInterfaceCapacity
	j.avroSet["input_interface_ip"] = j.InputInterfaceIP
	j.avroSet["output_interface_ip"] = j.OutputInterfaceIP
	j.avroSet["custom_str"] = j.CustomStr
	j.avroSet["custom_int"] = j.CustomInt
	j.avroSet["custom_bigint"] = j.CustomBigInt
	j.avroSet["eventType"] = j.EventType
	j.hasSetAvro = true
	return j.avroSet
}

func (j *JCHF) SetMap() {
	j.avroSet = map[string]interface{}{}
}