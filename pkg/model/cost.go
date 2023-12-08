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

package model

import (
	"time"

	"github.com/SENERGY-Platform/billing/pkg/opencost"
)

type CostEntry struct {
	Cpu                      float64 `json:"cpu"`
	Ram                      float64 `json:"ram"`
	Storage                  float64 `json:"storage"`
	opencost.AllocationEntry `json:"-"`
}

type CostOverview = map[CostType]CostEntry

type CostOverviewEntries = map[CostType]CostEntry

type CostType = string

const CostTypeAnalytics CostType = "analytics"

type CostContainers = map[string]CostEntry

type CostContainerEntries = map[string]CostEntry

type CostControllers = map[string]CostEntry

type CostControllerEntries = map[string]CostEntry

type CostWithChildren struct {
	Month    CostEntry                   `json:"month"`
	Children map[string]CostWithChildren `json:"children"`
}

type CostTree map[string]CostWithChildren

type UserCostTree map[string]CostTree

type BillingInformation struct {
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	CreatedAt time.Time `json:"created_at"`
	UserId    string    `json:"-"`
	Tree      CostTree  `json:"tree"`
}
