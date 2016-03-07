// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"fmt"

	"github.com/juju/errors"
	"gopkg.in/mgo.v2/txn"

	"github.com/juju/juju/network"
)

// ipAddressDoc describes the persistent state of an IP address assigned to a
// link-layer network device (a.k.a network interface card - NIC).
type ipAddressDoc struct {
	// DocID is the IP address ID, prefixed by ModelUUID.
	DocID string `bson:"_id"`

	// ID is the ID of the IP address, which is generated from a sequence like
	// for machines and units.
	ID string `bson:"id"`

	// ModelUUID is the UUID of the model this IP address belongs to.
	ModelUUID string `bson:"model-uuid"`

	// ProviderID is a provider-specific ID of the IP address, prefixed by
	// ModelUUID. Empty when not supported by the provider.
	ProviderID string `bson:"providerid,omitempty"`

	// DeviceName is the name of the link-layer device this IP address is
	// assigned to.
	DeviceName string `bson:"device-name"`

	// MachineID is the ID of the machine this IP address's device belongs to.
	MachineID string `bson:"machine-id"`

	// SubnetID is the ID of the subnet this IP address belongs to. Must be
	// empty for ConfigType LoopbackIPAddress.
	SubnetID string `bson:"subnet-id"`

	// ConfigMethod is the method used to configure this IP address.
	ConfigMethod AddressConfigMethod `bson:"config-method"`

	// Value is the value of the configured IP address, e.g. 192.168.1.2 or
	// 2001:db8::/64.
	Value string `bson:"value"`

	// DNSServers contains a list of DNS nameservers that apply to this IP
	// address's device. Can be empty.
	DNSServers []string `bson:"dns-servers,omitempty"`

	// DNSSearchDomains contains a list of DNS domain names used to qualify
	// hostnames, and can be empty.
	DNSSearchDomains []string `bson:"dns-search-domains,omitempty"`

	// GatewayAddress is the IP address of the gateway this IP address's device
	// uses. Can be empty.
	GatewayAddress string `bson:"gateway-address,omitempty"`
}

// AddressConfigMethod is the method used to configure a link-layer device's IP
// address.
type AddressConfigMethod string

const (
	// LoopbackAddress is used for IP addresses of LoopbackDevice types.
	LoopbackAddress AddressConfigMethod = "loopback"

	// StaticAddress is used for statically configured addresses.
	StaticAddress AddressConfigMethod = "static"

	// DynamicAddress is used for addresses dynamically configured via DHCP.
	DynamicAddress AddressConfigMethod = "dynamic"

	// ManualAddress is used for manually configured addresses.
	ManualAddress AddressConfigMethod = "manual"
)

// IsValidAddressConfigMethod returns whether the given value is a valid method
// to configure a link-layer network device's IP address.
func IsValidAddressConfigMethod(value string) bool {
	switch AddressConfigMethod(value) {
	case LoopbackAddress, StaticAddress, DynamicAddress, ManualAddress:
		return true
	}
	return false
}

// Address represents the state of an IP address assigned to a link-layer
// network device on a machine.
//
// TODO(dimitern): Rename to IPAddress once the IPAddress type is gone
// along with the addressable containers handling code?
type Address struct {
	st  *State
	doc ipAddressDoc
}

func newIPAddress(st *State, doc ipAddressDoc) *Address {
	return &Address{st: st, doc: doc}
}

// DocID returns the globally unique ID of the IP address, including the model
// UUID as prefix.
func (addr *Address) DocID() string {
	return addr.st.docID(addr.doc.DocID)
}

// ID returns the Juju-generated unique ID of the IP address.
func (addr *Address) ID() string {
	return addr.doc.ID
}

// ProviderID returns the provider-specific IP address ID, if set.
func (addr *Address) ProviderID() network.Id {
	return network.Id(addr.localProviderID())
}

func (addr *Address) localProviderID() string {
	return addr.st.localID(addr.doc.ProviderID)
}

// MachineID returns the ID of the machine this IP address belongs to.
func (addr *Address) MachineID() string {
	return addr.doc.MachineID
}

// Machine returns the Machine this IP address belongs to.
func (addr *Address) Machine() (*Machine, error) {
	return addr.st.Machine(addr.doc.MachineID)
}

// machineProxy is a convenience wrapper for calling Machine methods from an
// *Address.
func (addr *Address) machineProxy() *Machine {
	return &Machine{st: addr.st, doc: machineDoc{Id: addr.doc.MachineID}}
}

// DeviceName returns the name of the link-layer device this IP address is
// assigned to.
func (addr *Address) DeviceName() string {
	return addr.doc.DeviceName
}

// Device returns the LinkLayeyDevice this IP address is assigned to.
func (addr *Address) Device() (*LinkLayerDevice, error) {
	return addr.machineProxy().LinkLayerDevice(addr.doc.DeviceName)
}

func (addr *Address) deviceDocID() string {
	deviceGlobalKey := addr.deviceGlobalKey()
	if deviceGlobalKey == "" {
		return ""
	}
	return addr.st.docID(deviceGlobalKey)
}

func (addr *Address) deviceGlobalKey() string {
	return linkLayerDeviceGlobalKey(addr.doc.MachineID, addr.doc.DeviceName)
}

// SubnetID returns the ID of the subnet this IP address comes from. For a
// LoopbackAddress, the subnet is always empty.
func (addr *Address) SubnetID() string {
	return addr.doc.SubnetID
}

// Subnet returns the Subnet this IP address comes from. Returns nil and no
// error for a LoopbackAddress.
func (addr *Address) Subnet() (*Subnet, error) {
	if addr.doc.SubnetID == "" {
		return nil, nil
	}

	return addr.st.Subnet(addr.doc.SubnetID)
}

// ConfigMethod returns the AddressConfigMethod used for this IP address.
func (addr *Address) ConfigMethod() AddressConfigMethod {
	return addr.doc.ConfigMethod
}

// Value returns the value of this IP address.
func (addr *Address) Value() string {
	return addr.doc.Value
}

// DNSServers returns the list of DNS nameservers to use, which can be empty.
func (addr *Address) DNSServers() []string {
	return addr.doc.DNSServers
}

// DNSSearchDomains returns the list of DNS domains to use for qualifying
// hostnames. Can be empty.
func (addr *Address) DNSSearchDomains() []string {
	return addr.doc.DNSSearchDomains
}

// GatewayAddress returns the gateway address to use, which can be empty.
func (addr *Address) GatewayAddress() string {
	return addr.doc.GatewayAddress
}

// String returns a human-readable representation of the IP address.
func (addr *Address) String() string {
	return fmt.Sprintf("%s address %q", addr.doc.ConfigMethod, addr.doc.Value)
}

func (addr *Address) globalKey() string {
	return ipAddressGlobalKey(addr.doc.ID)
}

func ipAddressGlobalKey(addressID string) string {
	if addressID == "" {
		return ""
	}
	return "ip#" + addressID
}

// Remove removes the IP address, if it exists. No error is returned when the
// address was already removed.
func (addr *Address) Remove() (err error) {
	defer errors.DeferredAnnotatef(&err, "cannot remove %s", addr)

	removeOp := removeIPAddressDocOp(addr.doc.DocID)
	return addr.st.runTransaction([]txn.Op{removeOp})
}

// removeIPAddressDocOpOp returns an operation to remove the ipAddressDoc
// matching the given ipAddressDocID, without asserting it still exists.
func removeIPAddressDocOp(ipAddressDocID string) txn.Op {
	return txn.Op{
		C:      ipAddressesC,
		Id:     ipAddressDocID,
		Remove: true,
	}
}

// insertIPAddressDocOp returns an operation inserting the given newDoc,
// asserting it does not exist yet.
func insertIPAddressDocOp(newDoc *ipAddressDoc) txn.Op {
	return txn.Op{
		C:      ipAddressesC,
		Id:     newDoc.DocID,
		Assert: txn.DocMissing,
		Insert: *newDoc,
	}
}
