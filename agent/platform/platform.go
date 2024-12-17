// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package platform contains platform specific utilities.
package platform

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aws/amazon-ssm-agent/agent/log"
)

const (
	gettingPlatformDetailsMessage = "getting platform details"
	notAvailableMessage           = "NotAvailable"
	commandOutputMessage          = "Command output %v"

	//map keys for cached platform data
	platformNameKey    = "platform_name"
	platformTypeKey    = "platform_type"
	platformVersionKey = "platform_version"
	platformSkuKey     = "platform_sku"
)

var (
	getPlatformDataFn = getPlatformData
	cache             = InitCache(time.Hour.Milliseconds())
)

// IsPlatformWindowsServer2012OrEarlier represents whether it is Windows 2012 and earlier or not
func IsPlatformWindowsServer2012OrEarlier(log log.T) (bool, error) {
	return isPlatformWindowsServer2012OrEarlier(log)
}

// IsPlatformWindowsServer2025OrLater returns true if current platform is Windows Server 2025 or later
func IsPlatformWindowsServer2025OrLater(log log.T) (bool, error) {
	return isPlatformWindowsServer2025OrLater(log)
}

// IsWindowsServer2025OrLater returns true if passed platformVersion is the same as of Windows Server 2025 or later
func IsWindowsServer2025OrLater(platformVersion string, log log.T) (bool, error) {
	return isWindowsServer2025OrLater(platformVersion, log)
}

func IsPlatformNanoServer(log log.T) (bool, error) {
	return isPlatformNanoServer(log)
}

// PlatformName gets the OS specific platform name.
func PlatformName(log log.T) (name string, err error) {
	// get cached value if exists
	if platformName, found := cache.Get(platformNameKey); found {
		return platformName, nil
	}

	return retrievePlatformName(log)
}

func retrievePlatformName(log log.T) (string, error) {
	if platformData, err := initPlatformDataCache(log); err != nil {
		return platformData.Name, err
	} else {
		name := platformData.Name
		platformName := ""
		for i := range name {
			runeVal, _ := utf8.DecodeRuneInString(name[i:])
			if runeVal == utf8.RuneError {
				// runeVal = rune(value[i]) - using this will convert \xa9 to valid unicode code point
				continue
			}
			platformName = platformName + fmt.Sprintf("%c", runeVal)
		}

		return platformName, nil
	}
}

// PlatformVersion gets the OS specific platform version.
func PlatformVersion(log log.T) (version string, err error) {
	// get cached value if exists
	if platformVersion, found := cache.Get(platformVersionKey); found {
		return platformVersion, nil
	}

	// cache platform data
	platformData, err := initPlatformDataCache(log)
	return platformData.Version, err
}

// PlatformSku gets the OS specific platform SKU number
func PlatformSku(log log.T) (string, error) {
	// get cached value if exists
	if platformSku, found := cache.Get(platformSkuKey); found {
		return platformSku, nil
	}

	// cache platform data
	platformData, err := initPlatformDataCache(log)
	return platformData.Sku, err
}

// PlatformType gets the OS specific platform type.
func PlatformType(log log.T) string {
	// get cached value if exists
	if platformType, found := cache.Get(platformTypeKey); found {
		return platformType
	}

	// cache platform data
	platformData, _ := initPlatformDataCache(log)
	return platformData.Type
}

func initPlatformDataCache(log log.T) (platformData PlatformData, err error) {
	if platformData, err = getPlatformDataFn(log); err == nil {
		cache.Put(platformNameKey, platformData.Name)
		cache.Put(platformVersionKey, platformData.Version)
		cache.Put(platformSkuKey, platformData.Sku)
		cache.Put(platformTypeKey, platformData.Type)
	} else {
		log.Warnf("Failed to get platform data: %v", err)
	}

	return platformData, err
}

func GetSystemInfo(log log.T, paramKey string) (string, error) {
	// get cached value if exists
	if systemInfo, found := cache.Get(paramKey); found {
		return systemInfo, nil
	}

	// cache system info
	return initSystemInfoCache(log, paramKey)
}

// Hostname of the computer.
func Hostname(log log.T) string {
	return fullyQualifiedDomainName(log)
}

// IP of the network interface
func IP() (selected string, err error) {
	var interfaces []net.Interface
	if interfaces, err = net.Interfaces(); err == nil {
		interfaces = filterInterface(interfaces)
		sort.Sort(byIndex(interfaces))
		candidates := make([]net.IP, 0)
		for _, i := range interfaces {
			var addrs []net.Addr
			if addrs, err = i.Addrs(); err != nil {
				continue
			}
			for _, addr := range addrs {
				switch v := addr.(type) {
				case *net.IPAddr:
					candidates = append(candidates, v.IP.To4())
					candidates = append(candidates, v.IP.To16())
				case *net.IPNet:
					candidates = append(candidates, v.IP.To4())
					candidates = append(candidates, v.IP.To16())
				}
			}
		}

		selectedIp, err := selectIp(candidates)
		if err == nil {
			selected = selectedIp.String()
		}
	} else {
		err = fmt.Errorf("failed to load network interfaces: %v", err)
	}

	if err != nil {
		err = fmt.Errorf("failed to determine IP address: %v", err)
	}

	return
}

// Selects a single IP address to be reported for this instance.
func selectIp(candidates []net.IP) (result net.IP, err error) {
	for _, ip := range candidates {
		if ip != nil && !ip.IsUnspecified() {
			if result == nil {
				result = ip
			} else if isLoopbackOrLinkLocal(result) {
				// Prefer addresses that are not loopbacks or link-local
				if !isLoopbackOrLinkLocal(ip) {
					result = ip
				}
			} else if !isLoopbackOrLinkLocal(ip) {
				// Among addresses that are not loopback or link-local, prefer IPv4
				if !isIpv4(result) && isIpv4(ip) {
					result = ip
				}
			}
		}
	}

	if result == nil {
		err = fmt.Errorf("no IP addresses found")
	}

	return
}

func isLoopbackOrLinkLocal(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func isIpv4(ip net.IP) bool {
	return ip.To4() != nil
}

// filterInterface removes interface that's not up or is a loopback/p2p
func filterInterface(interfaces []net.Interface) (i []net.Interface) {
	for _, v := range interfaces {
		if (v.Flags&net.FlagUp != 0) && (v.Flags&net.FlagLoopback == 0) && (v.Flags&net.FlagPointToPoint == 0) {
			i = append(i, v)
		}
	}
	return
}

// byIndex implements sorting for net.Interface.
type byIndex []net.Interface

func (b byIndex) Len() int           { return len(b) }
func (b byIndex) Less(i, j int) bool { return b[i].Index < b[j].Index }
func (b byIndex) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func ClearCache() {
	cache.Flush()
}

type PlatformData struct {
	Name    string
	Version string
	Sku     string
	Type    string
}

type PlatformCache struct {
	//cached data
	data map[string]*PlatformCacheItem
	//time to live for cache data in milliseconds
	ttl int64
	//synced access to data
	lock sync.Mutex
}

type PlatformCacheItem struct {
	value string
	//time in milliseconds when the data was saved to cache
	cachedTime int64
}

func InitCache(ttl int64) *PlatformCache {
	return &PlatformCache{
		data: make(map[string]*PlatformCacheItem),
		ttl:  ttl,
	}
}

func (cache *PlatformCache) Put(k, v string) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cacheItem := &PlatformCacheItem{value: v, cachedTime: time.Now().UnixMilli()}
	cache.data[k] = cacheItem
}

func (cache *PlatformCache) Get(k string) (v string, found bool) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if cacheItem, hit := cache.data[k]; hit {
		if time.Now().UnixMilli()-cacheItem.cachedTime < cache.ttl {
			v = cacheItem.value
			found = true
		} else {
			delete(cache.data, k) //remove stale cache data
		}
	}
	return
}

func (cache *PlatformCache) Flush() {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.data = make(map[string]*PlatformCacheItem)
}
