package main

import (
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"log/syslog"

	"github.com/osrg/gobgp/table"
)

var (
	Log *syslog.Writer
)

const (
	IP = iota
	AS
	INV
)

// Initialise log files
func InitLog(infoHandle, errorHandle io.Writer) {
	var err error

	Log, err = syslog.New(syslog.LOG_NOTICE, "goasmap")
	if err != nil {
		log.Fatal(err)
	}
}

// Check if string (str) is present in array of string (list)
func stringInSlice(str string, list []string) bool {
	for _, lstr := range list {
		if lstr == str {
			return true
		}
	}
	return false
}

// Takes an IP in reverse (rip) then inverse it and fill it out with 0s
// Return a list of Ases containing said IP
func getAs4(rip []string) []string {
	var ases []string
	var ip string
	var i int

	for i = len(rip) - 1; i >= 0; i-- {
		ip += rip[i] + "."
	}
	for i = len(rip); i < 4; i++ {
		ip += "0."
	}
	ip = ip[:len(ip)-1]
	if debug {
		Log.Debug("IPv4 request received: " + ip + "/32")
	}
	if _, _, err := net.ParseCIDR(ip + "/32"); err != nil {
		Log.Warning("Invalid IPv4")
		return nil
	}
	ASList.s.RLock()
	defer ASList.s.RUnlock()
	for s, v, ok := ASList.RPfx.LongestPrefix(table.CidrToRadixkey(ip + "/32")); ok == true && len(s) > 0; s, v, ok = ASList.RPfx.LongestPrefix(s[:len(s)-1]) {
		data := v.([]string)
		if len(data[0][2:]) > 2 {
			_, ok = ASList.As[data[0]]
			if ok {
				GetAsn(data[0][2:])
			}

			res := data[0][2:] + " | " + data[1]
			_, ok = ASList.DelegAs[data[0][2:]]
			if ok {
				res += " | " + ASList.DelegAs[data[0][2:]].CC + " | " + ASList.DelegAs[data[0][2:]].Rir + " | " + ASList.DelegAs[data[0][2:]].Date
			}
			res += " | " + ASList.As[data[0]].ASName
			ases = append(ases, res)
		}
	}
	return ases
}

// Takes an IP in reverse (rip) then inverse it and fill it out with 0s
// Return a list of ASes containing said IP
func getAs6(rip []string) []string {
	var ases []string
	var ip string
	var i int

	lrip := len(rip)
	subnet := strconv.Itoa(lrip * 4)
	for i = lrip - 1; i >= 0; i-- {
		ip += rip[i]
		if (lrip-i)%4 == 0 {
			ip += ":"
		}
	}
	for i = lrip; i%4 != 0; i++ {
		ip += "0"
	}
	if lrip < 32 {
		ip += ":"
		if lrip%4 != 0 {
			ip += ":"
		}
	} else {
		ip = ip[:len(ip)-1]
	}
	if debug {
		Log.Debug("IPv6 received: " + ip + "/" + subnet)
	}
	if _, _, err := net.ParseCIDR(ip + "/" + subnet); err != nil {
		Log.Warning("Invalid IPv6")
		return nil
	}
	ASList.s.RLock()
	defer ASList.s.RUnlock()
	for s, v, ok := ASList.RPfx6.LongestPrefix(table.CidrToRadixkey(ip + "/" + subnet)); ok == true && len(s) > 0; s, v, ok = ASList.RPfx6.LongestPrefix(s[:len(s)-1]) {
		data := v.([]string)
		if len(data[0][2:]) > 2 {
			_, ok = ASList.As[data[0]]
			if ok {
				GetAsn(data[0][2:])
			}

			res := data[0][2:] + " | " + data[1]
			_, ok = ASList.DelegAs[data[0][2:]]
			if ok {
				res += " | " + ASList.DelegAs[data[0][2:]].CC + " | " + ASList.DelegAs[data[0][2:]].Rir + " | " + ASList.DelegAs[data[0][2:]].Date
			}
			res += " | " + ASList.As[data[0]].ASName
			ases = append(ases, res)
		}
	}
	return ases
}

// Return a list of prefix that an AS (as) has
func getPfx(as string) []string {
	var ans []string

	if len(as) > 2 {
		ASList.s.RLock()
		defer ASList.s.RUnlock()
		if isKnownAs(as[2:]) {
			upperAs := strings.ToUpper(as)

			GetAsn(upperAs[2:])

			res := upperAs[2:]
			_, ok := ASList.DelegAs[upperAs[2:]]
			if ok {
				res += " | " + ASList.DelegAs[upperAs[2:]].CC + " | " + ASList.DelegAs[upperAs[2:]].Rir + " | " + ASList.DelegAs[upperAs[2:]].Date
			}
			res += " | " + ASList.As[upperAs].ASName
			ans = append(ans, res)
			return ans
		}
	}
	return nil
}
