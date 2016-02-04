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

var testUrl = map[string]DomainType{
	"www.google.com":  domainTypeProxy,
	"www.youtube.com": domainTypeProxy,
	"www.twitter.com": domainTypeProxy,
	"i.ytimg.com":     domainTypeProxy,
	"www.baidu.com":   domainTypeDirect,
	"weibo.com":       domainTypeDirect,
}

func BenchmarkRegexp(b *testing.B) {
	list := newDomainList()
	list.initDomainList("testdata/direct_test", domainTypeDirect)
	list.initDomainList("testdata/proxy_test", domainTypeProxy)

	for url, expected := range testUrl {
		result := list.regexJudge(url)
		if result != expected {
			b.Errorf("URL: %s misjudged to %v.\n", url, result)
		}
	}
}

func BenchmarkGoMap(b *testing.B) {
	list := newDomainList()
	list.initDomainList("testdata/direct_test", domainTypeDirect)
	list.initDomainList("testdata/proxy_test", domainTypeProxy)

	for url, expected := range testUrl {
		result := list.regexJudge(url)
		if result != expected {
			b.Errorf("URL: %s misjudged to %v.\n", url, result)
		}
	}
}

func (domainList *DomainList) oldJudge(url string) (domainType DomainType) {
	debug.Printf("judging host: %s", url)
	if domainList.Domain[url] == domainTypeReject {
		debug.Printf("host or domain should reject")
		return domainTypeReject
	}
	if parentProxy.empty() { // no way to retry, so always visit directly
		return domainTypeDirect
	}
	if url == "" { // simple host or private ip
		return domainTypeDirect
	}
	if domainList.Domain[url] == domainTypeDirect {
		debug.Printf("host or domain should direct")
		return domainTypeDirect
	}
	if domainList.Domain[url] == domainTypeProxy {
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
	if domainList.Domain[url] == domainTypeDirect {
		debug.Printf("host or domain should direct")
		return domainTypeDirect
	}
	if domainList.Domain[url] == domainTypeProxy {
		debug.Printf("host or domain should using proxy")
		return domainTypeProxy
	}

	return domainTypeProxy
}
