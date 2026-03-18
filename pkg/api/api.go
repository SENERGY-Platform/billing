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
	"context"
	"errors"
	"net/http"

	"slices"
	"strings"
	"sync"
	"time"

	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"github.com/SENERGY-Platform/billing/pkg/controller"
	"github.com/SENERGY-Platform/billing/pkg/log"
	"github.com/SENERGY-Platform/billing/pkg/model"
	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	"github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

var endpoints = []func(router *gin.Engine, config configuration.Config, controller *controller.Controller){}

type userQuery struct {
	ForUser string `form:"for_user"`
}

func Start(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, controller *controller.Controller) (err error) {
	log.Logger.Info("start api")
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin_mw.StructLoggerHandlerWithDefaultGenerators(
			log.Logger.With(attributes.LogRecordTypeKey, attributes.HttpAccessLogRecordTypeVal),
			attributes.Provider,
			[]string{},
			nil,
		),
		requestid.New(requestid.WithCustomHeaderStrKey("X-Request-ID")),
		gin_mw.ErrorHandler(model.GetStatusCode, ", "),
		gin_mw.StructRecoveryHandler(log.Logger, gin_mw.DefaultRecoveryFunc),
	)
	for _, endpoint := range endpoints {
		endpoint(router, config, controller)
	}
	server := &http.Server{Addr: ":" + config.ApiPort, Handler: router, WriteTimeout: 120 * time.Second, ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	wg.Add(1)
	go func() {
		log.Logger.Info("listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Logger.Error("api server error", attributes.ErrorKey, err)
			panic(err)
		}
	}()
	go func() {
		<-ctx.Done()
		err := server.Shutdown(context.Background())
		if err != nil {
			log.Logger.Error("api shutdown failed", attributes.ErrorKey, err)
		} else {
			log.Logger.Debug("api shutdown")
		}
		wg.Done()
	}()
	return nil
}

func getUserId(context *gin.Context) (string, error) {
	request := context.Request
	query := userQuery{}
	err := context.ShouldBindQuery(&query)
	if err != nil {
		return "", errors.Join(model.ErrBadRequest, err)
	}
	forUser := query.ForUser
	if forUser != "" {
		roles := strings.Split(request.Header.Get("X-User-Roles"), ", ")
		if !slices.Contains(roles, "admin") {
			return "", errors.Join(model.ErrForbidden, errors.New("forbidden"))
		}
		return forUser, nil
	}
	return request.Header.Get("X-UserId"), nil
}

func getToken(request *http.Request) (string, error) {
	return request.Header.Get("Authorization"), nil
}
