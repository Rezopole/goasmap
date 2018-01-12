package main

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var UnkAs []asRange
var UnkAs32 []asRange

type asRange struct {
	first int
	last  int
}

const ASLink = "https://www.iana.org/assignments/as-numbers/as-numbers-1.csv"
const AS32Link = "https://www.iana.org/assignments/as-numbers/as-numbers-2.csv"

func GetUnkAs() {
	resp, err := http.Get(ASLink)
	if err != nil {
		Log.Err(err.Error())
		return
	}
	defer resp.Body.Close()

	read := csv.NewReader(resp.Body)
	rec, err := read.Read()
	UnkAs = make([]asRange, 0)
	for rec, err = read.Read(); err == nil; rec, err = read.Read() {
		if !strings.Contains(rec[0], "-") {
			rec[0] = rec[0] + "-" + rec[0]
		}
		if len(rec[2]) <= 0 {
			rng := strings.Split(rec[0], "-")
			first, _ := strconv.Atoi(rng[0])
			last, _ := strconv.Atoi(rng[1])
			UnkAs = append(UnkAs, asRange{first: first, last: last})
		}
	}
}

func GetUnkAs32() {
	resp, err := http.Get(AS32Link)
	if err != nil {
		Log.Err(err.Error())
		return
	}
	defer resp.Body.Close()

	read := csv.NewReader(resp.Body)
	rec, err := read.Read()
	UnkAs32 = make([]asRange, 0)
	for rec, err = read.Read(); err == nil; rec, err = read.Read() {
		if !strings.Contains(rec[0], "-") {
			rec[0] = rec[0] + "-" + rec[0]
		}
		if len(rec[2]) <= 0 {
			rng := strings.Split(rec[0], "-")
			first, err := strconv.Atoi(rng[0])
			if err == nil {
				last, err := strconv.Atoi(rng[1])
				if err == nil {
					UnkAs32 = append(UnkAs, asRange{first: first, last: last})
				}
			}
		}
	}
}

func InitUnkAs() {
	for {
		ASList.s.Lock()
		GetUnkAs()
		GetUnkAs32()
		ASList.s.Unlock()
		time.Sleep(24 * time.Hour)
	}
}

func isKnownAs(as string) bool {
	asn, err := strconv.Atoi(as)
	if err != nil {
		return false
	}
	if asn > 65535 {
		for i := 0; i < len(UnkAs32); i++ {
			if UnkAs32[i].first >= asn && UnkAs32[i].last <= asn {
				return false
			}
		}
		return true
	}
	if asn > -1 && asn < 65536 {
		for i := 0; i < len(UnkAs); i++ {
			if UnkAs[i].first >= asn && UnkAs[i].last <= asn {
				return false
			}
		}
		return true
	}
	return false
}
