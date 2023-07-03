package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/vmware/go-vcloud-director/v2/govcd"
	"sigs.k8s.io/yaml"
)

type Config struct {
	Host     URL
	Insecure bool
	Token    string
	TokenOrg string

	Org string
	VDC string

	EdgeGateway string
	ExternalIP string
	InternalIP string
}

type URL struct {
	url.URL
}

func (u *URL) UnmarshalJSON(data []byte) error {
	var rawURL string
	err := json.Unmarshal(data, &rawURL)
	if err != nil {
		return err
	}
	url, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	u.URL = *url
	return nil
}

func (c *Config) Client() (*govcd.VCDClient, error) {
	endpoint := c.Host.JoinPath("api")
	vcdclient := govcd.NewVCDClient(*endpoint, c.Insecure)
	err := vcdclient.SetToken(c.TokenOrg, govcd.ApiTokenHeader, c.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %s", err)
	}
	return vcdclient, nil
}

func ConfigFromFile(path string) (*Config, error) {
	c := &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	err = yaml.UnmarshalStrict(data, c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}
	return c, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s CONFIG_FILE", os.Args[0])
	}
	configFilename := os.Args[1]

	config, err := ConfigFromFile(configFilename)
	if err != nil {
		log.Fatalf("failed to load configuration: %s", err)
	}

	client, err := config.Client() // We now have a client
	if err != nil {
		log.Fatalf("failed to create client: %s", err)
	}

	org, err := client.GetOrgByName(config.Org)
	if err != nil {
		log.Fatalf("failed to find org %q: %s", config.Org, err)
	}
	vdc, err := org.GetVDCByName(config.VDC, false)
	if err != nil {
		log.Fatalf("failed to find vdc %q: %s", config.VDC, err)
	}
	egw, err := vdc.GetEdgeGatewayByName(config.EdgeGateway, false)
	if err != nil {
		log.Fatalf("failed to find edge gateway %q: %s", config.EdgeGateway, err)
	}

	r := govcd.NatRule{
		Description: "test",
		Protocol: "ANY",
		ExternalIP:  config.ExternalIP,
		InternalIP:  config.InternalIP,
	}
	_, err = egw.AddDNATRule(r)
	if err != nil {
		log.Fatalf("failed to create DNAT rule: %s", err)
	}
}
