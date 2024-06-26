/*
 *    Copyright 2023 InfAI (CC SES)
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package opencost

import (
	"net/http"
	"sync"
	"time"

	"github.com/SENERGY-Platform/billing/pkg/configuration"
)

type cacheEntry struct {
	value      interface{}
	validUntil time.Time
}

type Client struct {
	config     configuration.Config
	cache      map[string]cacheEntry
	cacheMux   sync.Mutex
	httpClient http.Client
}

func NewClient(config configuration.Config) (*Client, error) {
	httpClient := http.Client{
		Timeout: 300 * time.Second,
	}
	return &Client{config: config, cache: map[string]cacheEntry{}, cacheMux: sync.Mutex{}, httpClient: httpClient}, nil
}
