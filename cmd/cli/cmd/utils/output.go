// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// OutputOptions encapsulates output format flags.
type OutputOptions struct {
	Output string
}

func (o *OutputOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "Output format. One of: (json, yaml, wide)")
}

func (o *OutputOptions) IsJSON() bool {
	return strings.EqualFold(o.Output, "json")
}

func (o *OutputOptions) IsYAML() bool {
	return strings.EqualFold(o.Output, "yaml")
}

func (o *OutputOptions) IsWide() bool {
	return strings.EqualFold(o.Output, "wide")
}

// PrintObject marshals an object to json or yaml if requested.
// Returns true if handled, false if tabular/default printing should proceed.
func (o *OutputOptions) PrintObject(out io.Writer, obj any) (bool, error) {
	if o.IsJSON() {
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return true, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = fmt.Fprintln(out, string(data))
		return true, err
	}
	if o.IsYAML() {
		data, err := yaml.Marshal(obj)
		if err != nil {
			return true, fmt.Errorf("failed to marshal YAML: %w", err)
		}
		_, err = fmt.Fprint(out, string(data))
		return true, err
	}
	return false, nil
}

// NewTabWriter creates a configured tabwriter.Writer.
func NewTabWriter(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, 6, 4, 3, ' ', 0)
}

// FormatAge formats creation timestamp into human readable age (e.g. 5m, 2h, 3d).
func FormatAge(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	duration := time.Since(timestamp.Time)
	if duration < 0 {
		duration = 0
	}
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}
