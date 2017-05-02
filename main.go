package main

import (
	"os"
	"sync"
	"flag"
	"strings"
	"strconv"
	
	"github.com/osrg/gobgp/config"
	"github.com/osrg/gobgp/table"
	api "github.com/osrg/gobgp/api"
	gobgp "github.com/osrg/gobgp/server"
	"github.com/spf13/viper"
	"github.com/coreos/go-systemd/daemon"	
	"github.com/armon/go-radix"
	"github.com/miekg/dns"
)

var ASList ASNS
var gconf Conf
var debug bool
var cfg string

type Conf struct {
	Dns dnsconf	`mapstructure:"dns"`
}

type dnsconf struct {
	Ip string	`mapstructure:"ip"`
	Port string	`mapstructure:"port"`
	Zone string	`mapstructure:"zone"`
	Max int32	`mapstructure:"max"`
	RRs []dns.RR
	MRRs map[string][]dns.RR
	curr int32
}

type Data struct {
	ASName string
	CC string
	Date string
	Prefix []string
}

type ASNS struct {
	As map[string]Data
	RPfx *radix.Tree
	RPfx6 *radix.Tree
	s  sync.RWMutex
}

// Remove prefix and as from list
func RemovePrefix(pfx string) {
	var v interface{}
	var ok bool
	if strings.Contains(pfx, ".") {
		v, ok = ASList.RPfx.Delete(table.CidrToRadixkey(pfx))
	} else {
		v, ok = ASList.RPfx6.Delete(table.CidrToRadixkey(pfx))
	}
	if !ok {
		return
	}
	_ = ok
	data := v.([]string)
	for i := range ASList.As[data[0]].Prefix {
		if ASList.As[data[0]].Prefix[i] == data[1] {
			tmp := ASList.As[data[0]]
			tmp.Prefix[i] = tmp.Prefix[len(tmp.Prefix)-1]
			tmp.Prefix = tmp.Prefix[:len(tmp.Prefix)-1]
			ASList.As[data[0]] = tmp
			break
		}
	}
}

// Starts BGP server in goroutin
// Add Neighbors to receive updates TODO add conf file
func StartBGP() *gobgp.BgpServer {
	var conf config.BgpConfigSet
	
        v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(cfg)
	err := v.ReadInConfig()
	if err != nil {
		Log.Err(err.Error())
	}
	err = v.Unmarshal(&conf)
	if err != nil {
		Log.Err(err.Error())
	}
	err = v.Unmarshal(&gconf)
	if err != nil {
		Log.Err(err.Error())
	}
	s := gobgp.NewBgpServer()
	go s.Serve()
	err = s.Start(&conf.Global)
	if err != nil {
		Log.Err(err.Error())
		return nil
	}
	for i := range conf.Neighbors {
		err = s.AddNeighbor(&conf.Neighbors[i])
		if err != nil {
			Log.Err(err.Error())
			return nil
		}
	}
	return s
}

func GetIPS4(v *viper.Viper) {
	ips := v.GetStringSlice("ipv4.ip")
	for idx := range ips {
		pfx := strings.Split(ips[idx], ",")
		rdx := table.CidrToRadixkey(pfx[0])
		ASList.RPfx.Insert(rdx, []string{"AS"+pfx[1], pfx[0]})
		if !stringInSlice(pfx[0], ASList.As["AS"+pfx[1]].Prefix) {
			ASList.s.Lock()
			tmp := ASList.As["AS"+pfx[1]]
			tmp.Prefix = append(ASList.As["AS"+pfx[1]].Prefix, pfx[0])
			ASList.As["AS"+pfx[1]] = tmp
			ASList.s.Unlock()
		}
	}
}

func GetIPS6(v *viper.Viper) {
	ips := v.GetStringSlice("ipv6.ip")
	for idx := range ips {
		pfx := strings.Split(ips[idx], ",")
		rdx := table.CidrToRadixkey(pfx[0])
		ASList.RPfx6.Insert(rdx, []string{"AS"+pfx[1], pfx[0]})
		if !stringInSlice(pfx[0], ASList.As["AS"+pfx[1]].Prefix) {
			ASList.s.Lock()
			tmp := ASList.As["AS"+pfx[1]]
			tmp.Prefix = append(ASList.As["AS"+pfx[1]].Prefix, pfx[0])
			ASList.As["AS"+pfx[1]] = tmp
			ASList.s.Unlock()
		}
	}
}

// Init all reserved IPv4 and IPv6 ni radix tree
func InitRadix() {	
	ASList.RPfx = radix.New()
	ASList.RPfx6 = radix.New()
	
        v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(cfg)
	if v.ReadInConfig() == nil {
		GetIPS4(v)
		GetIPS6(v)
	}
	inc := v.GetStringSlice("ipv4.include")
	for idx := range inc {
		v.SetConfigFile(inc[idx])
		if v.ReadInConfig() == nil {
			GetIPS4(v)
		}
	}
	inc = v.GetStringSlice("ipv6.include")
	for idx := range inc {
		v.SetConfigFile(inc[idx])
		if v.ReadInConfig() == nil {
			GetIPS6(v)
		}
	}
}

func main() {
	var as string
	ASList.As = make(map[string]Data)

	// Init log
	InitLog(os.Stdout, os.Stderr)
	
	// Parse flags
	flag.BoolVar(&debug, "debug", false, "Debug mode")
	flag.StringVar(&cfg, "f", "/etc/goasmap/goasmap.conf", "Config file")
	flag.Parse()

	// Init Radix
	InitRadix()
	if debug {
		Log.Debug("Radix tree initialized with custom IPv4 and IPv6")
	}
	
	s := StartBGP()
	if s == nil {
		return
	}
	if debug {
		Log.Debug("BGP server running")
	}
		
	// Init and Launch DNS
	udpdns := &dns.Server{Addr: gconf.Dns.Ip+":"+gconf.Dns.Port, Net: "udp"}
	go udpdns.ListenAndServe()
	dns.HandleFunc(".", handleRequest)
	tcpdns := &dns.Server{Addr: gconf.Dns.Ip+":"+gconf.Dns.Port, Net: "tcp"}
	go tcpdns.ListenAndServe()
	dns.HandleFunc(".", handleRequest)
	InitDnsZone()	
	if debug {
		Log.Debug("DNS server running")
	}

	// Start grpc Server
	grpcServer := api.NewGrpcServer(s, "127.0.0.1:50051")
	go func() {
		if err := grpcServer.Serve(); err != nil {
			Log.Err("failed to listen grpc port: "+err.Error())
			os.Exit(1)
		}
	}()
	
	// Send notification to systemd
	daemon.SdNotify(false, "NOTIFY_SOCKET")
	
	// Only listen to update messages
	w := s.Watch(gobgp.WatchUpdate(false))
	for {
		select {
		case ev := <-w.Event():
			// // Message is always an update message
			// // Can add a switch type just to be safe
			pathlist := ev.(*gobgp.WatchEventUpdate).PathList
			for _, path := range pathlist {
				pfx := path.GetNlri().String()
				if !path.IsWithdraw {
					if debug {
						Log.Debug("Learned prefix: "+pfx+". Nexthop: "+path.GetNexthop().String())
					}
					aslist := path.GetAsList()
					as = "AS"+strconv.FormatUint(uint64(aslist[len(aslist)-1]), 10)
					rdx := table.CidrToRadixkey(pfx)
					if strings.Contains(pfx, ".") {
						if _, ok := ASList.RPfx.Get(rdx); !ok {
							ASList.s.Lock()
							ASList.RPfx.Insert(rdx, []string{as, pfx})
							ASList.s.Unlock()
						}
					} else {
						if _, ok := ASList.RPfx6.Get(rdx); !ok {
							ASList.s.Lock()
							ASList.RPfx6.Insert(rdx, []string{as, pfx})
							ASList.s.Unlock()
						}

					}
					if !stringInSlice(pfx, ASList.As[as].Prefix) {
						ASList.s.Lock()
						tmp := ASList.As[as]
						tmp.Prefix = append(ASList.As[as].Prefix, pfx)
						ASList.As[as] = tmp
						ASList.s.Unlock()
					}
				} else {
					if debug {
 						Log.Debug("Withdrawn prefix: "+pfx+". Nexthop: "+path.GetNexthop().String())
					}
					ASList.s.Lock()
					RemovePrefix(pfx)
					ASList.s.Unlock()
				}
			}
		}
	}
}
