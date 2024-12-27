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

func TestGetInstancesInASGwithClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockASG := &MockAutoScalingClient{}

	type args struct {
		ctx     context.Context
		asgName string
	}
	tests := []struct {
		name    string
		args    args
		setup   func()
		want    []types.Instance
		wantErr bool
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
								Instances: []types.Instance{
									{
										InstanceId: aws.String("i-1234567890abcdef0"),
									},
								},
							},
						},
					}, nil
				}
			},
			want: []types.Instance{
				{
					InstanceId: aws.String("i-1234567890abcdef0"),
				},
			},
			wantErr: false,
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
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got, err := GetInstancesInASGwithClient(tt.args.ctx, tt.args.asgName, mockASG)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstancesInASGwithClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInstancesInASGwithClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}
