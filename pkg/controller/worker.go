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
	"log"

	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/SENERGY-Platform/billing/pkg/model"
)

func (c *Controller) StoreMonthlyBillingInformation(nMonths int) error {
	now := time.Now().UTC()

	usersOffset := 0
	userLimit := 50
	hasMoreUsers := true

	for hasMoreUsers {
		jwt, err := c.keycloakClient.LoginClient(context.Background(), c.config.KeycloakClient, c.config.KeycloakSecret, "master")
		if err != nil {
			return err
		}

		users, err := c.keycloakClient.GetUsers(context.Background(), jwt.AccessToken, "master", gocloak.GetUsersParams{
			First: &usersOffset,
			Max:   &userLimit,
		})
		if err != nil {
			return err
		}
		usersOffset += len(users)
		hasMoreUsers = len(users) == userLimit

		for _, user := range users {
			for i := 1; i <= nMonths; i++ {
				to := time.Date(now.Year(), now.Month()-time.Month(i-1), 1, 0, 0, 0, 0, time.UTC)
				from := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)

				log.Println("Fetching monthly billing information from " + from.Format(time.RFC3339) + " to " + to.Format(time.RFC3339))
				tree, err := c.calc.GetTree("Bearer "+jwt.AccessToken, true, &from, &to, user.ID)
				if err != nil {
					return err
				}
				log.Println("Storing tree for user " + *user.ID)
				billingInformation := model.BillingInformation{From: from, UserId: *user.ID, To: to, CreatedAt: now, Tree: tree}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err = c.db.SetBillingInformation(ctx, billingInformation)
				cancel()
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
