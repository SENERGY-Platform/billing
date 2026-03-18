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

package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"github.com/SENERGY-Platform/billing/pkg/controller"
	"github.com/SENERGY-Platform/billing/pkg/model"
	"github.com/gin-gonic/gin"
)

func init() {
	endpoints = append(endpoints, BillingComponentEndpoints)
}

type billingMonthPath struct {
	Year  int `uri:"year" binding:"required"`
	Month int `uri:"month" binding:"required"`
}

func BillingComponentEndpoints(router *gin.Engine, config configuration.Config, controller *controller.Controller) {
	router.GET("/billing-components", func(c *gin.Context) {
		userId, err := getUserId(c)
		if err != nil {
			c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		overview, err := controller.ListAvailableBillingInformation(userId)
		if err != nil {
			c.Error(errors.Join(model.GetError(http.StatusInternalServerError), err))
			return
		}
		c.JSON(http.StatusOK, overview)
	})

	router.GET("/billing-components/:year/:month", func(c *gin.Context) {
		userId, err := getUserId(c)
		if err != nil {
			c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		path := billingMonthPath{}
		err = c.ShouldBindUri(&path)
		if err != nil {
			c.Error(errors.Join(model.GetError(http.StatusBadRequest), err))
			return
		}
		overview, err := controller.GetBillingInformation(userId, time.Date(path.Year, time.Month(path.Month), 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			c.Error(errors.Join(model.GetError(http.StatusInternalServerError), err))
			return
		}
		c.JSON(http.StatusOK, overview)
	})

}
