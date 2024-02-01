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
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"github.com/SENERGY-Platform/billing/pkg/database"
	"github.com/SENERGY-Platform/billing/pkg/opencost"
)

type Controller struct {
	opencost *opencost.Client
	config   configuration.Config
	db       *database.Mongo

	prometheus v1.API
}

func NewController(ctx context.Context, conf configuration.Config, fatal func(err error), db *database.Mongo) (*Controller, error) {
	opencostClient, err := opencost.NewClient(conf)
	if err != nil {
		return nil, err
	}
	prometheusClient, err := api.NewClient(api.Config{
		Address: conf.PrometheusUrl,
	})
	if err != nil {
		return nil, err
	}
	controller := &Controller{
		opencost:   opencostClient,
		config:     conf,
		db:         db,
		prometheus: v1.NewAPI(prometheusClient),
	}
	if err != nil {
		return nil, err
	}

	return controller, nil
}
