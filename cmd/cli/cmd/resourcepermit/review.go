// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xhit/go-str2duration/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/retry"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api/resourcepermit"
)

const (
	denyValue    = "deny"
	approveValue = "approve"
)

type ReviewOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	Name         string
	Namespace    string
	Approve      bool
	Deny         bool
	NoColor      bool
	Message      string
	StartTimeStr string
	DurationStr  string
	KeepForStr   string

	Client ctrlclient.Client
}

func NewCmdReview(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &ReviewOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "review NAME [flags]",
		Short: "Review a ResourcePermit",
		Args:  cobra.ExactArgs(1),
		Example: `  # interactive review
  kubectl capsule resource-permit review grant-admin --namespace default

  # non-interactive approve/deny
  kubectl capsule resource-permit review grant-admin --namespace default --approve
  kubectl capsule resource-permit review grant-admin --namespace default --deny

  # review as another user with explicit groups
  kubectl capsule resource-permit review grant-admin --namespace default --approve \
    --as alice@example.com --as-group platform-engineers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Namespace of the ResourcePermit")
	cmd.Flags().BoolVar(&o.Approve, "approve", false, "Approve the request")
	cmd.Flags().BoolVar(&o.Deny, "deny", false, "Deny the request")
	cmd.Flags().BoolVar(&o.NoColor, "no-color", false, "Don't colorize the output")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "Optional review message")
	cmd.Flags().StringVar(&o.StartTimeStr, "start-time", "", "Start time (RFC3339 format, e.g. 2025-07-15T14:45:00Z)")
	cmd.Flags().StringVar(&o.DurationStr, "duration", "",
		"The ExtendedDuration this request is available for (e.g. 5m, 1h30m) [Overwrites the value from the request, if defined]")
	cmd.Flags().StringVar(&o.KeepForStr, "keep-for", "",
		"The ExtendedDuration this request is archived for (e.g. 5m, 1h30m) [Overwrites the value from the request, if defined]")

	return cmd
}

func (o *ReviewOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil && o.Factory != nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	if o.Namespace == "" && o.Factory != nil {
		ns, _, _ := o.Factory.Namespace()
		o.Namespace = ns
	}
	if o.Namespace == "" {
		o.Namespace = "default"
	}

	br := &v1beta2.ResourcePermit{}
	if err := o.Client.Get(
		ctx,
		ctrlclient.ObjectKey{Name: o.Name, Namespace: o.Namespace},
		br,
	); err != nil {
		return err
	}

	if br.Status.Phase == "" {
		return fmt.Errorf(
			"ResourcePermit %s is not yet processed, current phase: %q",
			o.Name,
			br.Status.Phase,
		)
	}

	if br.Status.Phase != v1beta2.ResourcePermitPhaseRequested {
		return fmt.Errorf(
			"ResourcePermit %s is not in Requested phase (already reviewed), current phase: %q",
			o.Name,
			br.Status.Phase,
		)
	}

	if br.Status.Request == nil {
		return fmt.Errorf("ResourcePermit %s has no prepared request properties", o.Name)
	}

	props := br.Status.Request.DeepCopy()

	if o.KeepForStr != "" {
		d, err := str2duration.ParseDuration(o.KeepForStr)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", o.KeepForStr, err)
		}

		keepFor := resourcepermit.ExtendedDuration(d)
		props.KeepFor = &keepFor
	}

	if o.DurationStr != "" {
		d, err := str2duration.ParseDuration(o.DurationStr)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", o.DurationStr, err)
		}

		props.Duration = &metav1.Duration{Duration: d}
	}

	if o.StartTimeStr != "" {
		var st metav1.Time
		if err := st.UnmarshalJSON(fmt.Appendf(nil, "%q", o.StartTimeStr)); err != nil {
			return fmt.Errorf("invalid start time %q: %w", o.StartTimeStr, err)
		}

		props.StartTime = &st
	}

	if o.Approve && o.Deny {
		return fmt.Errorf("--approve and --deny are mutually exclusive")
	}

	action := ""
	if o.Approve {
		action = approveValue
	} else if o.Deny {
		action = denyValue
	} else {
		printResourcePermitsApprovalTable(o.IOStreams.Out, br, props, !o.NoColor)

		reader := bufio.NewReader(o.IOStreams.In)
		for {
			_, _ = fmt.Fprint(o.IOStreams.Out, "Approve this request? [y/n]: ")

			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}

			input = strings.ToLower(strings.TrimSpace(input))
			if input == "y" {
				action = approveValue
				break
			} else if input == "n" {
				action = denyValue
				break
			} else {
				_, _ = fmt.Fprintln(o.IOStreams.Out, "Invalid input. Please type 'y' or 'n'.")
			}
		}
	}

	return retry.OnError(
		retry.DefaultRetry,
		apierrors.IsConflict,
		func() error {
			if err := o.Client.Get(
				ctx,
				ctrlclient.ObjectKey{Name: o.Name, Namespace: o.Namespace},
				br,
			); err != nil {
				return err
			}

			return patchResourcePermitStatus(ctx, o.Client, br, func() error {
				switch action {
				case approveValue:
					br.Status.Phase = v1beta2.ResourcePermitPhaseApproved
					br.Status.Request = props.DeepCopy()
				case denyValue:
					br.Status.Phase = v1beta2.ResourcePermitPhaseDenied
				default:
					return fmt.Errorf("unsupported review action %q", action)
				}

				if br.Status.Review == nil {
					br.Status.Review = &v1beta2.ReviewInfo{}
				}

				br.Status.Review.Message = o.Message

				return nil
			})
		},
	)
}
