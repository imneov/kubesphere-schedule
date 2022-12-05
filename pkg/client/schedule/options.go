/*
Copyright 2020 KubeSphere Authors

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

package schedule

import (
	"time"

	"github.com/spf13/pflag"

	"kubesphere.io/scheduling/pkg/utils/reflectutils"
)

type Options struct {
	ReleaseControllerOptions *ReleaseControllerOptions `json:"releaseControllerOptions,omitempty" yaml:"releaseControllerOptions,omitempty" mapstructure:"releaseControllerOptions"`
}

type ReleaseControllerOptions struct {
	MaxConcurrent int           `json:"maxConcurrent,omitempty" yaml:"maxConcurrent,omitempty" mapstructure:"maxConcurrent"`
	WaitTime      time.Duration `json:"waitTime,omitempty" yaml:"waitTime,omitempty" mapstructure:"waitTime"`
}

func NewOptions() *Options {
	return &Options{
		ReleaseControllerOptions: &ReleaseControllerOptions{
			MaxConcurrent: 10,
			WaitTime:      30 * time.Second,
		},
	}
}

// Validate check options values
func (s *Options) Validate() []error {
	var errors []error

	return errors
}

// ApplyTo overrides options if it's valid, which endpoint is not empty
func (s *Options) ApplyTo(options *Options) {

	if s.ReleaseControllerOptions != nil {
		reflectutils.Override(options, s)
	}
}

// AddFlags add options flags to command line flags,
func (s *Options) AddFlags(fs *pflag.FlagSet, c *Options) {
	// if s3-endpoint if left empty, following options will be ignored
	fs.DurationVar(&s.ReleaseControllerOptions.WaitTime, "schedule-release-controller-options-wait-time", c.ReleaseControllerOptions.WaitTime, "wait time when check release is ready or not")
	fs.IntVar(&s.ReleaseControllerOptions.MaxConcurrent, "schedule-release-controller-options-max-concurrent", c.ReleaseControllerOptions.MaxConcurrent, "the maximum number of concurrent Reconciles which can be run for release controller")
}
