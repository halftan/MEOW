package main

import (
	"net"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/cyfdecyf/bufio"
)

type DomainList struct {
	Domain map[string]DomainType

	Reject, Direct, Proxy []*regexp.Regexp
	sync.RWMutex
}

type DomainType byte

const (
	domainTypeUnknown DomainType = iota
	domainTypeDirect
	domainTypeProxy
	domainTypeReject
)

func newDomainList() *DomainList {
	return &DomainList{
		Domain: make(map[string]DomainType),
		Reject: make([]*regexp.Regexp, 0),
		Direct: make([]*regexp.Regexp, 0),
		Proxy:  make([]*regexp.Regexp, 0),
	}
}

func (domainList *DomainList) judge(url *URL) (domainType DomainType) {
	debug.Printf("judging host: %s", url.Host)
	if url.Domain == "" { // simple host or private ip
		return domainTypeDirect
	}
	hostString := strings.ToLower(url.Host)
	for _, regex := range domainList.Reject {
		if regex == nil {
			break
		}
		if regex.MatchString(hostString) {
			debug.Printf("host should be rejected")
			return domainTypeReject
		}
	}
	if parentProxy.empty() { // no way to retry, so always visit directly
		errl.Println("Parent proxy not configured! Bypassing request.")
		return domainTypeDirect
	}
	for _, regex := range domainList.Direct {
		if regex == nil {
			break
		}
		if regex.MatchString(hostString) {
			debug.Printf("host should bypass")
			return domainTypeDirect
		}
	}
	for _, regex := range domainList.Proxy {
		if regex == nil {
			break
		}
		if regex.MatchString(hostString) {
			debug.Printf("host should use proxy")
			return domainTypeProxy
		}
	}
	if domainList.Domain[url.Host] == domainTypeDirect || domainList.Domain[url.Domain] == domainTypeDirect {
		debug.Printf("host or domain should direct")
		return domainTypeDirect
	}
	if domainList.Domain[url.Host] == domainTypeProxy || domainList.Domain[url.Domain] == domainTypeProxy {
		debug.Printf("host or domain should using proxy")
		return domainTypeProxy
	}

	if !config.JudgeByIP {
		return domainTypeProxy
	}

	var ip string
	isIP, isPrivate := hostIsIP(url.Host)
	if isIP {
		if isPrivate {
			domainList.add(url.Host, domainTypeDirect)
			return domainTypeDirect
		}
		ip = url.Host
	} else {
		hostIPs, err := net.LookupIP(url.Host)
		if err != nil {
			errl.Printf("error looking up host ip %s, err %s", url.Host, err)
			return domainTypeProxy
		}
		ip = hostIPs[0].String()
	}

	if ipShouldDirect(ip) {
		domainList.add(url.Host, domainTypeDirect)
		return domainTypeDirect
	} else {
		domainList.add(url.Host, domainTypeProxy)
		return domainTypeProxy
	}
}

func (domainList *DomainList) add(host string, domainType DomainType) {
	domainList.Lock()
	defer domainList.Unlock()
	domainList.Domain[host] = domainType
}

func (domainList *DomainList) GetDomainList() []string {
	lst := make([]string, 0)
	for site, domainType := range domainList.Domain {
		if domainType == domainTypeDirect {
			lst = append(lst, site)
		}
	}
	return lst
}

var domainList = newDomainList()

func (domainList *DomainList) initDomainList(domainListFile string, domainType DomainType) {
	var err error
	if err = isFileExists(domainListFile); err != nil {
		return
	}
	f, err := os.Open(domainListFile)
	if err != nil {
		errl.Println("Error opening domain list:", err)
		return
	}
	defer f.Close()

	domainList.Lock()
	defer domainList.Unlock()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain == "" {
			continue
		}
		if domain[0] == '/' {
			// Regex domain config
			domain = strings.Trim(domain, "/")
			domainRegex, err := regexp.Compile(domain)
			if err != nil {
				errl.Printf("Invalid regexp %s", domain)
			}
			switch domainType {
			case domainTypeProxy:
				domainList.Proxy = append(domainList.Proxy, domainRegex)
			case domainTypeDirect:
				domainList.Direct = append(domainList.Direct, domainRegex)
			case domainTypeReject:
				domainList.Reject = append(domainList.Reject, domainRegex)
			}
			debug.Printf("Loaded regexp domain %s as type %v", domain, domainType)
		} else {
			domainList.Domain[domain] = domainType
			debug.Printf("Loaded domain %s as type %v", domain, domainType)
		}
	}
	if scanner.Err() != nil {
		errl.Printf("Error reading domain list %s: %v\n", domainListFile, scanner.Err())
	}
}
