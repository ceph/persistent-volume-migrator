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

package logger

import (
	"fmt"

	klog "k8s.io/klog/v2"
)

// ErrorLog helps in logging errors.
func ErrorLog(message string, args ...interface{}) {
	logMessage := fmt.Sprintf(message, args...)
	klog.ErrorDepth(0, logMessage)
}

// DefaultLog helps in logging with klog.level 0.
func DefaultLog(message string, args ...interface{}) {
	logMessage := fmt.Sprintf(message, args...)
	klog.InfoDepth(0, logMessage)
}
