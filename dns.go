package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/miekg/dns"
)

func InitDnsZone() {
	gconf.Dns.MRRs = make(map[string][]dns.RR)
	f, _ := os.Open(gconf.Dns.Zone)
	defer f.Close()
	read := bufio.NewReader(f)
	to := dns.ParseZone(read, "", "")
	for x := range to {
		gconf.Dns.MRRs[x.RR.Header().Name] = append(gconf.Dns.MRRs[x.RR.Header().Name], x.RR)
		gconf.Dns.RRs = append(gconf.Dns.RRs, x.RR)
	}
}

// handleRequest is called when receive a dns request
func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	var ans []string

	m := new(dns.Msg)
	m.SetReply(r)
	if atomic.LoadInt32(&gconf.Dns.curr) >= gconf.Dns.Max {
		w.WriteMsg(m)
		Log.Debug("Max number of simultaneous requests reached")
		return
	}
	if debug {
		Log.Debug(strconv.Itoa(int(atomic.LoadInt32(&gconf.Dns.curr))) + " DNS requests")
	}
	atomic.AddInt32(&gconf.Dns.curr, 1)
	if debug {
		Log.Debug("Received request: " + r.String())
	}
	if r.Question[0].Qtype == dns.TypeTXT {
		name := r.Question[0].Name

		// for idx := range gconf.Dns.RRs {
		for key := range gconf.Dns.MRRs {
			if strings.HasSuffix(name, "."+key) {
				for idx := range gconf.Dns.MRRs[key] {
					if gconf.Dns.MRRs[key][idx].Header().Rrtype == dns.TypeTXT {
						name = name[:strings.LastIndex(name, "."+key)]
						req := strings.Split(name, ".")
						switch req[len(req)-1] {
						case "origin":
							ans = getAs4(req[:len(req)-1])
							break
						case "origin6":
							ans = getAs6(req[:len(req)-1])
							break
						default:
							ans = getPfx(req[len(req)-1])
						}
						break
					}
				}
				break
			}
		}
		m.Answer = make([]dns.RR, len(ans))
		for i := 0; i < len(ans); i++ {
			m.Answer[i] = &dns.TXT{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0}, Txt: []string{ans[i]}}
		}
	} else {
		// for idx := range gconf.Dns.RRs {
		// if gconf.Dns.RRs[idx].Header().Rrtype == r.Question[0].Qtype && dns.CompareDomainName(r.Question[0].Name, gconf.Dns.RRs[idx].Header().Name) >= 3 {
		for idx := range gconf.Dns.MRRs[r.Question[0].Name] {
			if gconf.Dns.MRRs[r.Question[0].Name][idx].Header().Rrtype == r.Question[0].Qtype {
				m.Answer = append(m.Answer, gconf.Dns.MRRs[r.Question[0].Name][idx])
			}
		}
	}
	atomic.AddInt32(&gconf.Dns.curr, -1)
	w.WriteMsg(m)
}
