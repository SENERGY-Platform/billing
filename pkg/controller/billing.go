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
	"time"

	"github.com/SENERGY-Platform/billing/pkg/model"
)

func (this *Controller) GetBillingInformation(ctx context.Context, userId string, from time.Time) (trees []model.BillingInformation, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return this.db.GetBillingInformation(timeoutCtx, userId, from)
}

func (this *Controller) ListAvailableBillingInformation(ctx context.Context, userId string) (dates []time.Time, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return this.db.ListAvailableBillingInformation(timeoutCtx, userId)
}
