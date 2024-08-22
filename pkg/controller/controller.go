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

	gocloak "github.com/Nerzal/gocloak/v13"
	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"github.com/SENERGY-Platform/billing/pkg/database"
	"github.com/SENERGY-Platform/cost-calculator/pkg/client"
)

type Controller struct {
	calc           client.Client
	config         configuration.Config
	keycloakClient *gocloak.GoCloak
	db             *database.Mongo
}

func NewController(ctx context.Context, conf configuration.Config, fatal func(err error), db *database.Mongo) *Controller {
	calc := client.New(conf.CalculatorUrl)

	keycloakClient := gocloak.NewClient(conf.KeycloakUrl)

	controller := &Controller{
		calc:           calc,
		config:         conf,
		db:             db,
		keycloakClient: keycloakClient,
	}

	return controller
}
