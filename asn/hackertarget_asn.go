package hackertarget_asn

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/* Used with IP query
For example:
"1.1.1.1","13335","1.1.1.0/24","CLOUDFLARENET - Cloudflare, Inc., US"
*/
type ASNIP struct {
	IP     net.IP
	Number int
	CIDR   *net.IPNet
	Name   string
}

/* Used with AS query
For example:
"13335","CLOUDFLARENET - Cloudflare, Inc., US"
104.20.208.0/20
172.68.148.0/22
104.17.128.0/20
162.159.14.0/24
108.162.240.0/24
162.158.41.0/24
188.114.109.0/24
162.158.44.0/24
108.162.226.0/24
...
*/
type ASNInfo struct {
	Number int
	Name   string
	CIDRs  string
}

const (
	// API used for consumption
	URL = "https://api.hackertarget.com/aslookup/?q=%s"
	// Regexes used for parsing results header
	ASNHeader_regex = "(?:\"([^\"]+)\")+"
)
const (
	IPADDR = 1 << iota
	AS
	NONE
)

// GetDataType identifies type of data used for query and needed by parsing
func GetDataType(data string) int {
	if CheckIP(data) {
		return IPADDR
	}
	if CheckAS(data) {
		return AS
	}
	return NONE
}

// CheckIP returns true/false if IP address is valid
func CheckIP(ip string) bool {
	if i := net.ParseIP(ip); i == nil {
		return false
	}
	return true
}

// CheckAS returns true/false if ASN address is valid
func CheckAS(as string) bool {
	if as[:1] != "AS" {
		return false
	}
	return true
}

// Query performs a GET request to the API to get the necessary info about an
// IP address or AS
func Query(data string) (string, error) {
	t := &http.Transport{
		IdleConnTimeout: 10 * time.Second,
	}
	c := &http.Client{
		Transport: t,
	}

	url := fmt.Sprintf(URL, data)
	r, err := c.Get(url)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	return string(body), err
}

// GetIPInfo calls the query funcion and then the parsing function to fill the
// ASNIP struct with the necessary info
func (i *ASNIP) GetIPInfo(ip string) bool {
	i.IP = net.ParseIP(ip)
	resp, err := Query(ip)
	if err != nil {
		return false
	}
	if i.ParseASNIP(resp) == false {
		log.Printf("Error parsing ASN IP data.\n")
		return false
	}

	return true
}

// GetASInfo calls the query funcion and then the parsing function to fill the
// ASNInfo struct with the necessary info
// TODO: do something when ASN doesn't exists on API's database.
func (i *ASNInfo) GetASInfo(as string) bool {
	i.Number, _ = strconv.Atoi(as[2:]) // set AS number, skipping "AS"

	resp, err := Query(as)
	if err != nil {
		return false
	}

	if i.ParseASN(resp) == false {
		log.Printf("Error parsing ASN IP data.\n")
		return false
	}
	return true
}

// ParseASN parses data from API, from header to list of CIDRs
func (i *ASNInfo) ParseASN(r string) bool {
	var (
		//err error
		RE = regexp.MustCompile(ASNHeader_regex)
	)
	finds := RE.FindAllStringSubmatch(r, -1)
	if finds == nil {
		return false
	}

	i.Name = finds[1][1]                                                   // complete name
	i.CIDRs = strings.ReplaceAll(r[strings.Index(r, "\n")+1:], "\n", ", ") // CIDR list
	i.CIDRs = i.CIDRs[:len(i.CIDRs)-1]

	return true
}

// ParseASNIP parses data from API, only header because no other info is added.
func (i *ASNIP) ParseASNIP(r string) bool {
	var (
		err error
		RE  = regexp.MustCompile(ASNHeader_regex)
	)
	finds := RE.FindAllStringSubmatch(r, -1)
	if finds == nil {
		return false
	}

	i.Number, _ = strconv.Atoi(finds[1][1]) // asn number

	_, i.CIDR, err = net.ParseCIDR(finds[2][1]) // cidr info
	if err != nil {
		log.Printf("Error parsing CIDR data.")
	}

	i.Name = finds[3][1] // complete name
	return true
}
