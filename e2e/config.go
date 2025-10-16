package main

// TestProxies contains the list of available test proxies
var TestProxies = []string{
	"http://34.146.222.111:8080", // Tokyo proxy 1
	"http://35.221.118.95:8080",  // Tokyo proxy 2
	"http://34.142.205.26:8080",  // Singapore proxy
}

// GetDefaultProxy returns the first proxy as the default
func GetDefaultProxy() string {
	return TestProxies[0]
}

// GetProxySubset returns a subset of proxies for testing
func GetProxySubset(count int) []string {
	if count >= len(TestProxies) {
		return TestProxies
	}
	return TestProxies[:count]
}
