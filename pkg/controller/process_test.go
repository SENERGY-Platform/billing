/*
 *    Copyright 2024 InfAI (CC SES)
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

package controller

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestGetCostTree(t *testing.T) {
	t.Skip("experiment")
	t.Log("must be manually evaluated")
	t.Log("expects port forwarding to prometheus: kubectl port-forward -n cattle-monitoring-system service/prometheus-operated 9090:9090")
	t.Log("expects port forwarding to opencost: kubectl port-forward -n opencost service/opencost 9003:9003")

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.PrometheusUrl = "http://localhost:9090"
	config.OpencostUrl = "http://localhost:9003"

	ctrl, err := NewController(context.Background(), config, func(err error) {
		t.Fatal(err)
		return
	}, nil)
	if err != nil {
		t.Error(err)
		return
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	result, err := ctrl.getCostTrees(start, end)
	if err != nil {
		t.Error(err)
		return
	}

	file, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		t.Error(err)
		return
	}

	location := "/tmp/process_test_out_" + strconv.FormatInt(time.Now().Unix(), 10) + ".json"
	err = os.WriteFile(location, file, 0644)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(location)
}
