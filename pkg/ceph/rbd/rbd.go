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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

const (
	tmpKeyFileLocation   = "/tmp/csi/keys"
	tmpKeyFileNamePrefix = "keyfile-"
)

func storeKey(key string) (string, error) {
	tmpfile, err := ioutil.TempFile(tmpKeyFileLocation, tmpKeyFileNamePrefix)
	if err != nil {
		return "", fmt.Errorf("error creating a temporary keyfile: %w", err)
	}
	defer func() {
		if err != nil {
			// don't complain about unhandled error
			_ = os.Remove(tmpfile.Name())
		}
	}()

	if _, err = tmpfile.Write([]byte(key)); err != nil {
		return "", fmt.Errorf("error writing key to temporary keyfile: %w", err)
	}

	keyFile := tmpfile.Name()
	if keyFile == "" {
		err = fmt.Errorf("error reading temporary filename for key: %w", err)
		return "", err
	}

	if err = tmpfile.Close(); err != nil {
		return "", fmt.Errorf("error closing temporary filename: %w", err)
	}

	return keyFile, nil
}

func execCommand(command string, args []string) ([]byte, error) {
	// #nosec
	cmd := exec.Command(command, args...)
	return cmd.CombinedOutput()
}

// RenameVolume renames the volume with given name
func (r *Connection) RenameVolume(newImageName, oldImageName string) error {
	var output []byte

	args := []string{"rename", oldImageName, newImageName, "--pool", r.Pool, "--id", r.ID, "-m", r.Monitors, "--keyfile=" + r.KeyFile}

	if r.DataPool != "" {
		args = append(args, "--data-pool", r.DataPool)
	}
	output, err := execCommand("rbd", args)

	if err != nil {
		return fmt.Errorf("%w. failed to rename rbd image, command output: %s", err, string(output))
	}
	return nil
}

// RenameVolume renames the volume with given name
func (r *Connection) RemoveVolumeAdmin(Pool, imageName string) error {
	var output []byte

	// args := []string{"rm", imageName, "--pool", r.Pool, "--id", r.ID, "-m", r.Monitors, "--keyfile=" + r.KeyFile}
	args := []string{"-m", r.Monitors, "rm", imageName, "--pool", r.Pool, "-c", "/etc/ceph/ceph.conf"}
	if r.DataPool != "" {
		args = append(args, "--data-pool", r.DataPool)
	}
	output, err := execCommand("rbd", args)

	if err != nil {
		return fmt.Errorf("%w. failed to rename rbd image, command output: %s", err, string(output))
	}
	return nil
}
