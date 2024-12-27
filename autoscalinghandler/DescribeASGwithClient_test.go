package autoscalinghandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/golang/mock/gomock"
	"reflect"
)

// MockAutoScalingClient is a mock implementation of the autoscaling.Client interface.
type MockAutoScalingClient struct {
	autoscaling.Client
	mockDescribeAutoScalingGroups func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
}

func (m *MockAutoScalingClient) DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return m.mockDescribeAutoScalingGroups(ctx, params, optFns...)
}

func TestDescxribeASGwithClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockASG := &MockAutoScalingClient{}

	type args struct {
		ctx               context.Context
		asgName           string
		autoScalingGroups []types.AutoScalingGroup
	}
	tests := []struct {
		name  string
		args  args
		setup func()
		want  string
		want1 []types.AutoScalingGroup
	}{
		{
			name: "successful describe",
			args: args{
				ctx:     context.Background(),
				asgName: "test-asg",
			},
			setup: func() {
				mockASG.mockDescribeAutoScalingGroups = func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
					return &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []types.AutoScalingGroup{
							{
								AutoScalingGroupName: aws.String("test-asg"),
							},
						},
					}, nil
				}
			},
			want: "",
			want1: []types.AutoScalingGroup{
				{
					AutoScalingGroupName: aws.String("test-asg"),
				},
			},
		},
		{
			name: "error describing",
			args: args{
				ctx:     context.Background(),
				asgName: "test-asg",
			},
			setup: func() {
				mockASG.mockDescribeAutoScalingGroups = func(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
					return nil, fmt.Errorf("error")
				}
			},
			want:  "Failed to describe Auto Scaling group: error",
			want1: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got, got1 := DescribeASGwithClient(tt.args.ctx, tt.args.asgName, mockASG)
			if got != nil && got.Error() != tt.want {
				t.Errorf("DescribeASGwithClient() got = %v, want %v", got.Error(), tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("DescribeASGwithClient() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
