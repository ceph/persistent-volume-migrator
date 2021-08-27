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

type Connection struct {
	Monitors string
	ID       string
	KeyFile  string
	Pool     string
	DataPool string
}

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

func NewConnection(monitor, id, key, pool, datapool string) (*Connection, error) {
	keyfile, err := storeKey(key)
	if err != nil {
		return nil, err
	}
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
func (r *Connection) RemoveVolume(imageName string) error {
	var output []byte

	args := []string{"rm", imageName, "--pool", r.Pool, "--id", r.ID, "-m", r.Monitors, "--keyfile=" + r.KeyFile}

	if r.DataPool != "" {
		args = append(args, "--data-pool", r.DataPool)
	}
	output, err := execCommand("rbd", args)

	if err != nil {
		return fmt.Errorf("%w. failed to rename rbd image, command output: %s", err, string(output))
	}
	return nil
}
