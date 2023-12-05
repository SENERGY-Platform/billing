/*
 * Copyright 2020 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use mongo file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package database

import (
	"context"
	"slices"
	"time"

	"github.com/SENERGY-Platform/billing/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const useridFieldName = "UserId"
const fromFieldName = "From"
const createdAtFieldName = "CreatedAt"

var useridKey string
var fromKey string
var createdAtKey string

func (db *Mongo) initBillingInformation() (err error) {
	useridKey, err = getBsonFieldName(model.BillingInformation{}, useridFieldName)
	if err != nil {
		return err
	}
	fromKey, err = getBsonFieldName(model.BillingInformation{}, fromFieldName)
	if err != nil {
		return err
	}
	createdAtKey, err = getBsonFieldName(model.BillingInformation{}, createdAtFieldName)
	if err != nil {
		return err
	}

	collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollection)
	err = db.ensureCompoundIndex(collection, "userFromindex", true, false, useridKey, fromKey)
	if err != nil {
		return err
	}
	err = db.ensureCompoundIndex(collection, "userFromCreatedAtindex", true, true, useridKey, fromKey, createdAtKey)
	if err != nil {
		return err
	}
	return nil

}

func (db *Mongo) billingInformationCollection() *mongo.Collection {
	return db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollection)
}

func (db *Mongo) GetBillingInformation(ctx context.Context, userId string, from time.Time) (trees []model.BillingInformation, err error) {
	trees = []model.BillingInformation{}
	cursor, err := db.billingInformationCollection().Find(ctx, bson.M{useridKey: userId, fromKey: from}, &options.FindOptions{Sort: bson.M{fromKey: -1}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return trees, nil
		}
		return trees, err
	}
	err = cursor.All(ctx, &trees)
	if err != nil {
		return trees, err
	}
	return
}

func (db *Mongo) ListAvailableBillingInformation(ctx context.Context, userId string) (dates []time.Time, err error) {
	dates = []time.Time{}
	opt := options.Find().SetProjection(bson.M{fromKey: 1, "_id": 0}).SetSort(bson.M{createdAtKey: -1})

	cursor, err := db.billingInformationCollection().Find(ctx, bson.M{useridKey: userId}, opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return dates, nil
		}
		return dates, err
	}
	wrappedFroms := []WrappedFrom{}
	err = cursor.All(ctx, &wrappedFroms)
	if err != nil {
		return dates, err
	}
	for _, d := range wrappedFroms {
		dates = append(dates, d.From)
	}
	// make distinct entries
	// distinct in db does not work well with sort
	return slices.Compact(dates), err
}

func (db *Mongo) SetBillingInformation(ctx context.Context, billingInformation model.BillingInformation) error {
	_, err := db.billingInformationCollection().ReplaceOne(ctx, bson.M{useridKey: billingInformation.UserId, createdAtKey: billingInformation.CreatedAt, fromKey: billingInformation.From}, billingInformation, options.Replace().SetUpsert(true))
	return err
}

func (db *Mongo) RemoveInstance(ctx context.Context, userId string, from time.Time, createdAt time.Time) error {
	_, err := db.billingInformationCollection().DeleteOne(ctx, bson.M{userId: userId, createdAtKey: createdAt, fromKey: from})
	return err
}

type WrappedFrom struct {
	From time.Time
}
