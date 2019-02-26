/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package blb

import (
	"k8s.io/cloud-provider-baiducloud/pkg/cloud-sdk/bce"
)

// Endpoint contains all endpoints of BLB.
var Endpoint = map[string]string{
	"bj": "blb.bj.baidubce.com",
	"gz": "blb.gz.baidubce.com",
	"su": "blb.su.baidubce.com",
	"hk": "blb.hkg.baidubce.com",
	"bd": "blb.bd.baidubce.com",
}

// Config contains all options for BLB.
type Config struct {
	*bce.Config
}

// NewConfig returns BLB config
func NewConfig(config *bce.Config) *Config {
	return &Config{config}
}

// Client is the BLB client
type Client struct {
	*bce.Client
}

// NewBLBClient returns BLB client
func NewBLBClient(config *Config) *Client {
	bceClient := bce.NewClient(config.Config)
	return &Client{bceClient}
}

// GetURL generates the full URL of http request for BLB
func (c *Client) GetURL(version string, params map[string]string) string {
	host := c.Endpoint
	if host == "" {
		host = Endpoint[c.GetRegion()]
	}
	uriPath := version
	return c.Client.GetURL(host, uriPath, params)
}
