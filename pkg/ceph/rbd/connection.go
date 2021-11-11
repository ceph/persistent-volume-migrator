/*
Copyright © 2021 The Persistent-Volume-Migrator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbd

import (
	"os"

	logger "persistent-volume-migrator/pkg/log"
)

type Connection struct {
	Monitors string
	ID       string
	KeyFile  string
	Pool     string
	DataPool string
}

func NewConnection(monitor, id, key, pool, datapool string) (*Connection, error) {
	keyfile, err := storeKey(key)
	if err != nil {
		return nil, err
	}
	logger.DefaultLog("New connection arg monitors: %s, id: %s, keyfile: %s, pool: %s, datapool: %s", monitor, id, keyfile, pool, datapool)
	return &Connection{
		Monitors: monitor,
		ID:       id,
		KeyFile:  keyfile,
		Pool:     pool,
		DataPool: datapool,
	}, nil
}

func (c *Connection) Destroy() error {
	return os.Remove(c.KeyFile)
}
