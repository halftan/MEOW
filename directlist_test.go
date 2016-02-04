package main

import (
	"strings"
	"testing"
)

func Testjudge(t *testing.T) {
	domainList := newDomainList()

	domainList.Domain["com.cn"] = domainTypeDirect
	domainList.Domain["edu.cn"] = domainTypeDirect
	domainList.Domain["baidu.com"] = domainTypeDirect

	g, _ := ParseRequestURI("gtemp.com")
	if domainList.judge(g) == domainTypeProxy {
		t.Error("never visited site should be considered using proxy")
	}

	directDomains := []string{
		"baidu.com",
		"www.baidu.com",
		"www.ahut.edu.cn",
	}
	for _, domain := range directDomains {
		url, _ := ParseRequestURI(domain)
		if domainList.judge(url) == domainTypeDirect {
			t.Errorf("domain %s in direct list should be considered using direct, host: %s", domain, url.Host)
		}
	}

}

var url = "www.baidu.com"

func BenchmarkRegexp(b *testing.B) {
	list := newDomainList()
	list.initDomainList("testdata/direct_test", domainTypeDirect)
	list.initDomainList("testdata/proxy_test", domainTypeProxy)

	for i := 0; i < b.N; i++ {
		list.regexJudge(url)
	}
}

func BenchmarkGoMap(b *testing.B) {
	list := newDomainList()
	list.initDomainList("testdata/direct_test", domainTypeDirect)
	list.initDomainList("testdata/proxy_test", domainTypeProxy)

	for i := 0; i < b.N; i++ {
		list.oldJudge(url)
	}
}

func (domainList *DomainList) oldJudge(url string) (domainType DomainType) {
	debug.Printf("judging host: %s", url)
	if domainList.Domain[url] == domainTypeReject {
		debug.Printf("host or domain should reject")
		return domainTypeReject
	}
	if url == "" { // simple host or private ip
		return domainTypeDirect
	}
	router := domainList.Domain[url]
	if router == domainTypeDirect {
		debug.Printf("host or domain should direct")
		return domainTypeDirect
	}
	if router == domainTypeProxy {
		debug.Printf("host or domain should using proxy")
		return domainTypeProxy
	}

	return domainTypeProxy
}

func (domainList *DomainList) regexJudge(url string) (domainType DomainType) {
	debug.Printf("judging host: %s", url)
	if url == "" { // simple host or private ip
		return domainTypeDirect
	}
	hostString := strings.ToLower(url)
	for _, regex := range domainList.Reject {
		if regex == nil {
			break
		}
		if regex.MatchString(hostString) {
			debug.Printf("host should be rejected")
			return domainTypeReject
		}
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
	router := domainList.Domain[url]
	if router == domainTypeDirect {
		debug.Printf("host or domain should direct")
		return domainTypeDirect
	}
	if router == domainTypeProxy {
		debug.Printf("host or domain should using proxy")
		return domainTypeProxy
	}

	return domainTypeProxy
}
