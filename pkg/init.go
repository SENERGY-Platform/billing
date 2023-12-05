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

package pkg

import (
	"context"

	"github.com/SENERGY-Platform/billing/pkg/api"
	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"github.com/SENERGY-Platform/billing/pkg/controller"
	"github.com/SENERGY-Platform/billing/pkg/database"

	"sync"
)

func Start(ctx context.Context, cancel context.CancelFunc, config configuration.Config, fatal func(err error)) (wg *sync.WaitGroup, err error) {
	wg = &sync.WaitGroup{}

	db, err := database.New(config, ctx, wg)
	if err != nil {
		return wg, err
	}

	ctrl, err := controller.NewController(ctx, config, fatal, db)
	if err != nil {
		return wg, err
	}

	if config.Job {
		err = ctrl.StoreMonthlyBillingInformation(config.JobMonths)
		if err != nil {
			return wg, err
		}
	}

	if config.Server {
		err = api.Start(ctx, wg, config, ctrl)
		if err != nil {
			return wg, err
		}
	} else {
		cancel()
	}

	return
}
