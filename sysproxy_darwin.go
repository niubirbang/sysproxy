//go:build darwin

package sysproxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/textproto"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	DefaultIgnores = []string{
		"127.0.0.1",
		"192.168.0.0/16",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"localhost",
		"*.local",
		"*.crashlytics.com",
		"<local>",
	}
)

func init() {
	// fmt.Println("sysproxy use darwin")
}

func OffAll() error {
	if err := OffHttps(); err != nil {
		return err
	}
	if err := OffHttp(); err != nil {
		return err
	}
	if err := OffSocks(); err != nil {
		return err
	}
	return nil
}

func SetIgnore(ignores []string) error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	return set("proxybypassdomains", s, ignores...)
}

func ClearIgnore() error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	return set("proxybypassdomains", s, "")
}

func GetIgnore() ([]string, error) {
	s, err := getNetworkInterface()
	if err != nil {
		return nil, err
	}
	m, err := get("proxybypassdomains", s)
	if err != nil {
		return nil, err
	}
	m = strings.TrimSpace(m)
	ignores := strings.Split(m, "\n")
	if len(ignores) != 0 && ignores[len(ignores)-1] == "" {
		ignores = ignores[:len(ignores)-1]
	}
	return ignores, nil
}

func OnHttps(addr Addr) error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	err = set("securewebproxy", s, addr.Host, strconv.Itoa(addr.Port))
	if err != nil {
		return err
	}
	err = set("securewebproxystate", s, "on")
	if err != nil {
		return err
	}
	return nil
}

func OffHttps() error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	err = set("securewebproxystate", s, "off")
	if err != nil {
		return err
	}
	return nil
}

func GetHttps() (*Addr, error) {
	s, err := getNetworkInterface()
	if err != nil {
		return nil, err
	}
	buf, err := get("securewebproxy", s)
	if err != nil {
		return nil, err
	}
	reader := textproto.NewReader(bufio.NewReader(bytes.NewBufferString(buf)))
	header, err := reader.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if header.Get("Enabled") == "Yes" {
		return ParseAddrPtr(fmt.Sprintf("%s:%s", header.Get("Server"), header.Get("Port"))), nil
	}
	return nil, nil
}

func OnHttp(addr Addr) error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	err = set("webproxy", s, addr.Host, strconv.Itoa(addr.Port))
	if err != nil {
		return err
	}
	err = set("webproxystate", s, "on")
	if err != nil {
		return err
	}
	return nil
}

func OffHttp() error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	err = set("webproxystate", s, "off")
	if err != nil {
		return err
	}
	return nil
}

func GetHttp() (*Addr, error) {
	s, err := getNetworkInterface()
	if err != nil {
		return nil, err
	}
	buf, err := get("webproxy", s)
	if err != nil {
		return nil, err
	}
	reader := textproto.NewReader(bufio.NewReader(bytes.NewBufferString(buf)))
	header, err := reader.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if header.Get("Enabled") == "Yes" {
		return ParseAddrPtr(fmt.Sprintf("%s:%s", header.Get("Server"), header.Get("Port"))), nil
	}
	return nil, nil
}

func OnSocks(addr Addr) error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	err = set("socksfirewallproxy", s, addr.Host, strconv.Itoa(addr.Port))
	if err != nil {
		return err
	}
	err = set("socksfirewallproxystate", s, "on")
	if err != nil {
		return err
	}
	return nil
}

func OffSocks() error {
	s, err := getNetworkInterface()
	if err != nil {
		return err
	}
	err = set("socksfirewallproxystate", s, "off")
	if err != nil {
		return err
	}
	return nil
}

func GetSocks() (*Addr, error) {
	s, err := getNetworkInterface()
	if err != nil {
		return nil, err
	}
	buf, err := get("socksfirewallproxy", s)
	if err != nil {
		return nil, err
	}
	reader := textproto.NewReader(bufio.NewReader(bytes.NewBufferString(buf)))
	header, err := reader.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if header.Get("Enabled") == "Yes" {
		return ParseAddrPtr(fmt.Sprintf("%s:%s", header.Get("Server"), header.Get("Port"))), nil
	}
	return nil, nil
}

func set(key string, inter string, value ...string) error {
	_, err := command("networksetup", append([]string{"-set" + key, inter}, value...)...)
	return err
}

func get(key string, inter string) (string, error) {
	return command("networksetup", "-get"+key, inter)
}

func getNetworkInterface() (string, error) {
	buf, err := command("sh", "-c", "networksetup -listnetworkserviceorder | grep -B 1 $(route -n get default | grep interface | awk '{print $2}')")
	if err != nil {
		return "", err
	}
	reader := textproto.NewReader(bufio.NewReader(bytes.NewBufferString(buf)))
	reg := regexp.MustCompile(`^\(\d+\)\s(.*)$`)
	device, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	match := reg.FindStringSubmatch(device)
	if len(match) <= 1 {
		return "", fmt.Errorf("unable to get network interface")
	}
	return match[1], nil
}

func command(name string, arg ...string) (string, error) {
	c := exec.Command(name, arg...)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%q: %w: %q", strings.Join(append([]string{name}, arg...), " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
