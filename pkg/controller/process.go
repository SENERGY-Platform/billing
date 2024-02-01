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
	"fmt"
	"github.com/SENERGY-Platform/billing/pkg/model"
	"github.com/SENERGY-Platform/billing/pkg/opencost"
	prometheus_model "github.com/prometheus/common/model"
	"log"
	"math"
	"slices"
	"strings"
	"time"
)

func (c *Controller) GetProcessTree(controllers opencost.AllocationResponse, userId string, start time.Time, end time.Time) (result model.CostTree, err error) {
	processCost := model.CostWithChildren{
		Children: map[string]model.CostWithChildren{},
	}

	marshallerCostTotal := model.CostWithChildren{}
	processIoCostTotal := model.CostWithChildren{}

	userProcessFactor, err := c.getUserProcessFactor(userId, start, end)
	if err != nil {
		return result, err
	}

	for key, value := range controllers.Data[0] {
		if slices.Contains(c.config.ProcessCostSources, key) {
			nameParts := strings.Split(key, ":")
			name := nameParts[len(nameParts)-1]

			child := model.CostWithChildren{
				Month: model.CostEntry{
					Cpu:     value.CpuCost * userProcessFactor,
					Ram:     value.RamCost * userProcessFactor,
					Storage: value.PvCost * userProcessFactor,
				},
				Children: map[string]model.CostWithChildren{},
			}

			processCost.Children[name] = child

			processCost.Month.Cpu = processCost.Month.Cpu + child.Month.Cpu
			processCost.Month.Ram = processCost.Month.Ram + child.Month.Ram
			processCost.Month.Storage = processCost.Month.Storage + child.Month.Storage

			processDefinitionFactors, err := c.getProcessDefinitionFactors(key, userId, start, end)
			if err != nil {
				return result, err
			}
			for processDefinition, factor := range processDefinitionFactors {
				grandchild := model.CostWithChildren{
					Month: model.CostEntry{
						Cpu:     child.Month.Cpu * factor,
						Ram:     child.Month.Ram * factor,
						Storage: child.Month.Storage * factor,
					},
					Children: map[string]model.CostWithChildren{},
				}
				child.Children[processDefinition] = grandchild
			}
		}

		if slices.Contains(c.config.MarshallingCostSources, key) {
			marshallerCostTotal.Month.Cpu = marshallerCostTotal.Month.Cpu + value.CpuCost
			marshallerCostTotal.Month.Ram = marshallerCostTotal.Month.Ram + value.RamCost
			marshallerCostTotal.Month.Storage = marshallerCostTotal.Month.Storage + value.PvCost
		}

		if slices.Contains(c.config.ProcessIoCostSources, key) {
			processIoCostTotal.Month.Cpu = processIoCostTotal.Month.Cpu + value.CpuCost
			processIoCostTotal.Month.Ram = processIoCostTotal.Month.Ram + value.RamCost
			processIoCostTotal.Month.Storage = processIoCostTotal.Month.Storage + value.PvCost
		}
	}

	processMarshallerFactor, err := c.getProcessMarshallerFactor(start, end)
	if err != nil {
		return result, err
	}

	userMarshallerFactor, err := c.getUserMarshallerFactor(userId, start, end)
	if err != nil {
		return result, err
	}

	marshallerCostProcesses := model.CostEntry{
		Cpu:     marshallerCostTotal.Month.Cpu * processMarshallerFactor,
		Ram:     marshallerCostTotal.Month.Ram * processMarshallerFactor,
		Storage: marshallerCostTotal.Month.Storage * processMarshallerFactor,
	}

	marshallerCostUser := model.CostWithChildren{
		Month: model.CostEntry{
			Cpu:     marshallerCostProcesses.Cpu * userMarshallerFactor,
			Ram:     marshallerCostProcesses.Ram * userMarshallerFactor,
			Storage: marshallerCostProcesses.Storage * userMarshallerFactor,
		},
		Children: map[string]model.CostWithChildren{},
	}

	processCost.Children["marshalling"] = marshallerCostUser

	userProcessIoFactor, err := c.getUserProcessIoFactor(userId, start, end)
	if err != nil {
		return result, err
	}

	processIoCostUser := model.CostWithChildren{
		Month: model.CostEntry{
			Cpu:     processIoCostTotal.Month.Cpu * userProcessIoFactor,
			Ram:     processIoCostTotal.Month.Ram * userProcessIoFactor,
			Storage: processIoCostTotal.Month.Storage * userProcessIoFactor,
		},
		Children: map[string]model.CostWithChildren{},
	}

	processCost.Children["process-io"] = processIoCostUser

	result = map[string]model.CostWithChildren{"process": processCost}
	return result, nil
}

func (c *Controller) getUserProcessFactor(userId string, start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.UserProcessCostFractionQuery, userId, start, end)
}

func (c *Controller) getProcessMarshallerFactor(start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.ProcessMarshallerCostFractionQuery, "", start, end)
}

func (c *Controller) getUserMarshallerFactor(userId string, start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.UserMarshallerCostFractionQuery, userId, start, end)
}

func (c *Controller) getUserProcessIoFactor(userId string, start time.Time, end time.Time) (float64, error) {
	return c.getValueFromPrometheus(c.config.UserProcessIoCostFractionQuery, userId, start, end)
}

func (c *Controller) getProcessDefinitionFactors(processCostSource string, userId string, start time.Time, end time.Time) (map[string]float64, error) {
	result := map[string]float64{}

	instanceId, ok := c.config.ProcessCostSourceToInstanceIdPlaceholderForProcessDefCostFraction[processCostSource]
	if !ok {
		return result, nil
	}

	query := strings.ReplaceAll(c.config.UserProcessDefinitionCostFractionQuery, "$instance_id", instanceId)

	ingreases, err := c.getValueMapFromPrometheus(query, userId, start, end)
	if err != nil {
		return result, err
	}
	sum := 0.0
	for _, e := range ingreases {
		sum = sum + e
	}
	for k, e := range ingreases {
		result[k] = e / sum
	}
	return result, nil
}

func (c *Controller) getValueFromPrometheus(query string, userId string, start time.Time, end time.Time) (float64, error) {
	query = strings.ReplaceAll(query, "$user_id", userId)
	query = strings.ReplaceAll(query, "$__range", end.Sub(start).Round(time.Second).String())
	resp, w, err := c.prometheus.Query(context.Background(), query, end)
	if err != nil {
		return 1, err
	}
	if len(w) > 0 {
		log.Printf("WARNING: prometheus warnings = %#v\n", w)
	}
	if resp.Type() != prometheus_model.ValScalar {
		return 1, fmt.Errorf("unexpected prometheus response %#v", resp)
	}
	value, ok := resp.(*prometheus_model.Scalar)
	if !ok {
		return 1, fmt.Errorf("unexpected prometheus response %#v", resp)
	}
	return sampleToFloat(value.Value), nil
}

func (c *Controller) getValueMapFromPrometheus(query string, userId string, start time.Time, end time.Time) (map[string]float64, error) {
	result := map[string]float64{}
	query = strings.ReplaceAll(query, "$user_id", userId)
	query = strings.ReplaceAll(query, "$__range", end.Sub(start).Round(time.Second).String())
	resp, w, err := c.prometheus.Query(context.Background(), query, end)
	if err != nil {
		return result, err
	}
	if len(w) > 0 {
		log.Printf("WARNING: prometheus warnings = %#v\n", w)
	}
	if resp.Type() != prometheus_model.ValVector {
		return result, fmt.Errorf("unexpected prometheus response %#v", resp)
	}
	values, ok := resp.(prometheus_model.Vector)
	if !ok {
		return result, fmt.Errorf("unexpected prometheus response %#v", resp)
	}
	for _, element := range values {
		label := ""
		for _, metricLabel := range element.Metric {
			label = string(metricLabel)
		}
		result[label] = sampleToFloat(element.Value)
	}
	return result, nil
}

func sampleToFloat(value prometheus_model.SampleValue) float64 {
	temp := float64(value)
	if math.IsNaN(temp) {
		return 0
	}
	return temp
}
