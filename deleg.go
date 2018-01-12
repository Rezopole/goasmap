package main

import (
	"bufio"
	"net/http"
	"strings"
	"sync"
	"time"
)

type SDeleg struct {
	CC   string
	Date string
	Rir  string
}

var delegArin = "http://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest"
var delegRipe = "http://ftp.ripe.net/ripe/stats/delegated-ripencc-latest"
var delegAfrinic = "http://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-latest"
var delegApnic = "http://ftp.apnic.net/pub/stats/apnic/delegated-apnic-latest"
var delegLacnic = "http://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-latest"

var delegMutex sync.Mutex
var delegWait sync.WaitGroup
var tmpDeleg map[string]SDeleg

var inFormat = "20060102"
var outFormat = "2006-01-02"

func readLine(deleg, pfx string) {
	var l []byte
	var err error
	var sTime string

	resp, err := http.Get(deleg)
	if err != nil {
		Log.Err(err.Error())
		return
	}
	read := bufio.NewReader(resp.Body)
	for l, _, err = read.ReadLine(); err == nil && !strings.HasPrefix(string(l), pfx); l, _, err = read.ReadLine() {
	}
	for ; err == nil; l, _, err = read.ReadLine() {
		f := strings.Split(string(l), "|")
		if len(f) >= 6 && f[2] == "asn" {
			t, err := time.Parse(inFormat, f[5])
			if err == nil {
				sTime = t.Format(outFormat)
			}
			delegMutex.Lock()
			tmpDeleg[f[3]] = SDeleg{CC: f[1], Date: sTime, Rir: pfx}
			delegMutex.Unlock()
		}
	}
	delegWait.Done()
}

func UpdateDelegation() {
	ASList.s.Lock()
	ASList.DelegAs = make(map[string]SDeleg)
	ASList.s.Unlock()
	for {
		ASList.s.Lock()
		tmpDeleg = ASList.DelegAs
		ASList.s.Unlock()
		delegWait.Add(5)
		go readLine(delegArin, "arin")
		go readLine(delegRipe, "ripe")
		go readLine(delegAfrinic, "afrinic")
		go readLine(delegApnic, "apnic")
		go readLine(delegLacnic, "lacnic")
		delegWait.Wait()
		ASList.s.Lock()
		ASList.DelegAs = tmpDeleg
		ASList.s.Unlock()
		time.Sleep(24 * time.Hour)
	}
}
