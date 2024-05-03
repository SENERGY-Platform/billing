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

package controller

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SENERGY-Platform/billing/pkg/model"
	"github.com/SENERGY-Platform/billing/pkg/opencost"
)

func (c *Controller) getCostOverview(from, to time.Time) (allocation opencost.AllocationResponse, err error) {
	allocation, err = c.opencost.Allocation(&opencost.AllocationOptions{
		Window:    from.Format(time.RFC3339) + "," + to.Format(time.RFC3339),
		Aggregate: "label:user,namespace",
	})
	if err != nil {
		return allocation, err
	}
	err = validateAllocation(&allocation)
	if err != nil {
		return allocation, err
	}
	return allocation, err
}

func (c *Controller) getCostContainers(from, to time.Time) (allocation opencost.AllocationResponse, err error) {
	allocation, err = c.opencost.Allocation(&opencost.AllocationOptions{
		Window:    from.Format(time.RFC3339) + "," + to.Format(time.RFC3339),
		Aggregate: "label:user,namespace,controller,container",
	})
	if err != nil {
		return allocation, err
	}
	err = validateAllocation(&allocation)
	if err != nil {
		return allocation, err
	}
	return allocation, err
}

func (c *Controller) getCostControllers(from, to time.Time) (allocation opencost.AllocationResponse, err error) {
	allocation, err = c.opencost.Allocation(&opencost.AllocationOptions{
		Window:    from.Format(time.RFC3339) + "," + to.Format(time.RFC3339),
		Aggregate: "label:user,namespace,controller",
	})
	if err != nil {
		return allocation, err
	}
	err = validateAllocation(&allocation)
	if err != nil {
		return allocation, err
	}
	return allocation, err
}

func (c *Controller) getCostTrees(from, to time.Time) (res model.UserCostTree, err error) {
	var overview, controllers, containers opencost.AllocationResponse
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		var temperr error
		overview, temperr = c.getCostOverview(from, to)
		if temperr != nil {
			err = temperr
		}
	}()
	go func() {
		defer wg.Done()
		var temperr error
		controllers, temperr = c.getCostControllers(from, to)
		if temperr != nil {
			err = temperr
		}
	}()
	go func() {
		defer wg.Done()
		var temperr error
		containers, temperr = c.getCostContainers(from, to)
		if temperr != nil {
			err = temperr
		}
	}()
	wg.Wait()
	if err != nil {
		return res, err
	}

	res = model.UserCostTree{}
	for overviewKey, overviewValue := range overview.Data[0] {
		overviewKeyParts := strings.Split(overviewKey, "/")
		if len(overviewKeyParts) != 2 {
			return res, errors.New("unexpected key " + overviewKey)
		}
		userId := overviewKeyParts[0]
		namespace := overviewKeyParts[1]

		userTree, ok := res[userId]
		if !ok {
			userTree = model.CostTree{}
		}

		controllerTree := model.CostTree{}
		for controllerKey, controllerValue := range controllers.Data[0] {
			controllerKeyParts := strings.Split(controllerKey, "/")
			if len(controllerKeyParts) != 3 {
				return res, errors.New("unexpected key " + controllerKey)
			}
			controllerUserId := controllerKeyParts[0]
			controllerNamespace := controllerKeyParts[1]
			controllerName := controllerKeyParts[2]

			if controllerUserId != userId || controllerNamespace != namespace {
				continue
			}

			containerTree := model.CostTree{}
			for containerKey, containerValue := range containers.Data[0] {
				containerKeyParts := strings.Split(containerKey, "/")
				if len(containerKeyParts) != 4 {
					return res, errors.New("unexpected key " + containerKey)
				}
				containerUserId := containerKeyParts[0]
				containerNamespace := containerKeyParts[1]
				containerControllerName := containerKeyParts[2]
				containerName := containerKeyParts[3]

				if containerUserId != userId || containerNamespace != namespace || containerControllerName != controllerName {
					continue
				}

				containerTree[containerName] = model.CostWithChildren{
					Month: model.CostEntry{
						Cpu:             containerValue.CpuCost,
						Ram:             containerValue.RamCost,
						Storage:         containerValue.PvCost,
						AllocationEntry: containerValue,
					},
				}
			}

			controllerTree[controllerName] = model.CostWithChildren{
				Month: model.CostEntry{
					Cpu:             controllerValue.CpuCost,
					Ram:             controllerValue.RamCost,
					Storage:         controllerValue.PvCost,
					AllocationEntry: controllerValue,
				},
				Children: containerTree,
			}
		}
		userTree[namespace] = model.CostWithChildren{
			Month: model.CostEntry{
				Cpu:             overviewValue.CpuCost,
				Ram:             overviewValue.RamCost,
				Storage:         overviewValue.PvCost,
				AllocationEntry: overviewValue,
			},
			Children: controllerTree,
		}

		processCost, err := c.GetProcessTree(controllers, userId, from, to)
		if err != nil {
			return res, err
		}
		for key, value := range processCost {
			userTree[key] = value
		}

		res[userId] = userTree
	}
	return res, nil
}

func (c *Controller) StoreMonthlyBillingInformation(nMonths int) error {
	now := time.Now().UTC()
	for i := 1; i <= nMonths; i++ {
		to := time.Date(now.Year(), now.Month()-time.Month(i-1), 1, 0, 0, 0, 0, time.UTC)
		from := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)

		log.Println("Fetching monthly billing information from " + from.Format(time.RFC3339) + " to " + to.Format(time.RFC3339))
		trees, err := c.getCostTrees(from, to)
		if err != nil {
			return err
		}

		log.Println("Got " + strconv.Itoa(len(trees)) + " trees")

		for userid, tree := range trees {
			log.Println("Storing tree for user " + userid)
			billingInformation := model.BillingInformation{From: from, UserId: userid, To: to, CreatedAt: now, Tree: tree}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err = c.db.SetBillingInformation(ctx, billingInformation)
			cancel()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
