package inzure

import (
	"testing"
)

func TestPacketRoutesEquals(t *testing.T) {
	a := &PacketRoute{
		IPs:   createIPs("*"),
		Ports: createPorts("80"),
	}
	b := &PacketRoute{
		IPs:   createIPs("*"),
		Ports: createPorts("80"),
	}
	if !a.Equals(b) {
		t.Fatal("packet routes unequal with *")
	}
	b.IPs = createIPs("192.168.1.0", "10.0.0.1", "123.123.123.123")
	a.IPs = createIPs("192.168.1.0", "10.0.0.1", "123.123.123.123")
	if !a.Equals(b) {
		t.Fatal("packet routes unequal with noncontinuous range")
	}
	a.IPs = createIPs("10.0.0.0/8")
	b.IPs = createIPs("10.0.0.0/8")
	if !a.Equals(b) {
		t.Fatal("packet routes unequal with large IP ranges")
	}
	b.IPs = createIPs("192.168.1.0", "10.0.0.1", "123.123.123.123")
	a.IPs = createIPs("192.168.1.0", "10.0.0.1", "123.123.123.123")
	a.Ports = createPorts("0-20000")
	b.Ports = createPorts("0-20000")
	if !a.Equals(b) {
		t.Fatal("packet routes unequal with large port ranges")
	}
	b.Ports = createPorts("10", "23-56")
	a.Ports = createPorts("10", "23-56")
	if !a.Equals(b) {
		t.Fatal("packet routes unequal with port range")
	}
	a.Ports = createPorts("10")
	if a.Equals(b) {
		t.Fatal("unequal packetroutes are equal (10)")
	}
	a.Ports = createPorts("23-56")
	if a.Equals(b) {
		t.Fatal("unequal packetroutes are equal (23-56)")
	}
	a.IPs = createIPs("10.0.0.0/7")
	b.IPs = createIPs("10.0.0.0/8")
	if a.Equals(b) {
		t.Fatal("unequal packet routes equal with large IP ranges")
	}
	b.IPs = createIPs("192.168.1.0", "10.0.0.1", "123.123.123.123")
	a.IPs = createIPs("192.168.1.0", "10.0.0.1", "123.123.123.123")
	a.Ports = createPorts("0-20000")
	b.Ports = createPorts("0-20001")
	if a.Equals(b) {
		t.Fatal("unequal packet routes equal with large port ranges")
	}
}
