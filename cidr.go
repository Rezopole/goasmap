package main

import (
	"net"
)

// (?) Maybe add these functions tu utils.go (?)

// Check if an IP (ip) is in a subnet (sn)
func inSubnet(ip, sn string) bool {
	IP := net.ParseIP(ip)
	_, IPNet, _ := net.ParseCIDR(sn)
	return IPNet.Contains(IP)
}

// Check if an IP (ip) is in a list of subnets (sn)
func inSubnets(ip string, sn []string) bool {
	for idx := range sn {
		if inSubnet(ip, sn[idx]) {
			return true
		}
	}
	return false
}

// Retreive all subnets in a list of subnets (sn)
// that an IP (ip) belongs to
func getSubnets(ip string, sn []string) []string {
	var subnets []string
	for idx := range sn {
		if inSubnet(ip, sn[idx]) {
			subnets = append(subnets, sn[idx])
		}
	}
	return subnets
}
