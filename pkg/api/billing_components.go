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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"github.com/SENERGY-Platform/billing/pkg/controller"
	"github.com/julienschmidt/httprouter"
)

func init() {
	endpoints = append(endpoints, BillingComponentEndpoints)
}

func BillingComponentEndpoints(router *httprouter.Router, config configuration.Config, controller *controller.Controller) {
	router.GET("/billing-components", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		userId, err := getUserId(config, request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		overview, err := controller.ListAvailableBillingInformation(userId)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(writer).Encode(overview)
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	})

	router.GET("/billing-components/:year/:month", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		userId, err := getUserId(config, request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		year, err := strconv.Atoi(params.ByName("year"))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		month, err := strconv.Atoi(params.ByName("month"))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		overview, err := controller.GetBillingInformation(userId, time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(writer).Encode(overview)
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	})

}
