/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
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
	"crypto/rand"
	"encoding/hex"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/billing/pkg/configuration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestStartAuthenticates needs a throwaway server with access control; MONGO_AUTH_TEST_USER and
// MONGO_AUTH_TEST_PASSWORD are root credentials, used to create and remove the test users.
func TestStartAuthenticates(t *testing.T) {
	url, rootUser, rootPassword := os.Getenv("MONGO_AUTH_TEST_URL"), os.Getenv("MONGO_AUTH_TEST_USER"), os.Getenv("MONGO_AUTH_TEST_PASSWORD")
	if testing.Short() || url == "" || rootUser == "" || rootPassword == "" {
		t.Skip("needs MONGO_AUTH_TEST_URL, MONGO_AUTH_TEST_USER and MONGO_AUTH_TEST_PASSWORD, not in -short")
	}
	ctx := context.Background()
	root, err := mongo.Connect(ctx, options.Client().ApplyURI(url).SetAuth(options.Credential{Username: rootUser, Password: rootPassword, AuthSource: "admin"}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Disconnect(ctx) })

	suffix := randomHex(t)
	testDB, otherDB := "billing_auth_test_"+suffix, "billing_auth_other_"+suffix
	svcUser, svcPassword := "billing-test-"+suffix, randomHex(t)
	otherUser, otherPassword := "billing-other-"+suffix, randomHex(t)
	readUser, readPassword := "billing-read-"+suffix, randomHex(t)
	createUser(t, root, svcUser, svcPassword, "readWrite", testDB)
	createUser(t, root, otherUser, otherPassword, "readWrite", otherDB)
	createUser(t, root, readUser, readPassword, "read", testDB)
	passwords := []string{svcPassword, otherPassword, readPassword, rootPassword}

	config := func(user, password string) configuration.Config {
		return &configuration.ConfigStruct{
			MongoUrl:        url,
			MongoUser:       user,
			MongoPassword:   password,
			MongoAuthSource: "admin",
			MongoDatabase:   testDB,
			MongoCollection: "trees",
		}
	}

	t.Run("New with correct credentials", func(t *testing.T) {
		svcCtx, cancel := context.WithCancel(ctx)
		wg := &sync.WaitGroup{}
		db, err := New(config(svcUser, svcPassword), svcCtx, wg)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		if _, err = db.ListAvailableBillingInformation(ctx, "user"); err != nil {
			t.Errorf("query as the service user: %v", err)
		}
		cancel()
		wg.Wait()
		assertIndexes(t, root, testDB, "trees", "userFromindex", "userFromCreatedAtindex")
	})

	cases := []struct {
		name, user, password string
		wantErr              string
	}{
		{"correct credentials", svcUser, svcPassword, ""},
		{"no credentials", "", "", "mongo startup check failed: "},
		{"user of another database", otherUser, otherPassword, "mongo startup check failed: "},
		{"wrong password", svcUser, svcPassword + "-wrong", "mongo startup check failed: "},
		// listCollections passes, index creation does not.
		{"read-only user", readUser, readPassword, "Unauthorized"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conf := config(c.user, c.password)
			if err := validateConfig(conf); err != nil {
				t.Fatal(err)
			}
			pools := &poolCounter{}
			svcCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			wg := &sync.WaitGroup{}
			db, err := start(svcCtx, wg, conf, clientOptions(conf).SetPoolMonitor(pools.monitor()), 10*time.Second)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				cancel()
				wg.Wait()
				pools.assertAllClosed(t)
				return
			}
			if err == nil {
				db.Disconnect()
				t.Fatalf("expected an error containing %q", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("unexpected error: %v", err)
			}
			for _, pw := range passwords {
				if strings.Contains(err.Error(), pw) {
					t.Error("error text contains a password")
				}
			}
			pools.assertAllClosed(t)
			assertNoWaiters(t, wg)
		})
	}
}

// createUser registers the cleanup first, so a partly failed creation is removed as well.
func createUser(t *testing.T, root *mongo.Client, user, password, role, db string) {
	t.Helper()
	admin := root.Database("admin")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = admin.RunCommand(ctx, bson.D{{Key: "dropUser", Value: user}}).Err()
		_ = root.Database(db).Drop(ctx)
	})
	cmd := bson.D{
		{Key: "createUser", Value: user},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: bson.A{bson.D{{Key: "role", Value: role}, {Key: "db", Value: db}}}},
	}
	if err := admin.RunCommand(context.Background(), cmd).Err(); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func assertIndexes(t *testing.T, root *mongo.Client, db, collection string, want ...string) {
	t.Helper()
	specs, err := root.Database(db).Collection(collection).Indexes().ListSpecifications(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, s := range specs {
		names = append(names, s.Name)
	}
	for _, w := range want {
		if !slices.Contains(names, w) {
			t.Errorf("index %q missing, have %v", w, names)
		}
	}
}

func randomHex(t *testing.T) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}
