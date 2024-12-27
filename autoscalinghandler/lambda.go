package autoscalinghandler

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	asgTypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
)

// ASGInfoLambda - Lambda function to get the availability zones for a given instance type
func ASGInfoLambda(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	log.Printf("ASGInfoLambda(%#v, %#v)", ctx, event)
	lc, _ := lambdacontext.FromContext(ctx)
	log.Printf("AWSRequestID: %#v", lc.AwsRequestID)

	asgNameInterface, ok := event.ResourceProperties["ASG"].(interface{})
	if !ok {
		err := fmt.Errorf("ASG property is missing or invalid")
		log.Printf("Error: %v", err)
		return "", nil, err
	}

	asgName, ok := asgNameInterface.(string)
	if !ok {
		err := fmt.Errorf("ASG property is not a string")
		log.Printf("Error: %v", err)
		return "", nil, err
	}

	instances, err := GetInstancesInASG(ctx, asgName)
	if err != nil {
		log.Printf("Error getting instances: %v", err)
		return "", nil, err
	}
	log.Printf("Found %d instances", len(instances))

	// Put the instance IDs into a slice
	instanceIdsSlice := make([]string, 0, len(instances))
	instancePublicIPssSlice := make([]string, 0, len(instances))
	//instancePublicIPv6sSlice := make([]string, 0, len(instances))
	for _, instance := range instances {
		if instance.State != nil && instance.State.Name == ec2Types.InstanceStateNameRunning {
			log.Printf("Instance: %#v", instance)
			instanceIdsSlice = append(instanceIdsSlice, *instance.InstanceId)
			instancePublicIPssSlice = append(instancePublicIPssSlice, *instance.PublicIpAddress)
			//instancePublicIPv6sSlice = append(instancePublicIPv6sSlice, *instance.Ipv6Address)
		}
	}
	log.Printf("For map: %d, %d", len(instanceIdsSlice), len(instancePublicIPssSlice))

	physicalResourceID := instanceIdsSlice[0]

	data := map[string]interface{}{
		"InstanceIds": instanceIdsSlice,
		"PublicIPs":   instancePublicIPssSlice,
		//"IPv6Addresses": instancePublicIPv6sSlice,
	}

	switch event.RequestType {
	case "Create":
		// No additional action needed for Create
	case "Delete":
		err = RemoveResources(event)
		if err != nil {
			log.Printf("Error removing resources: %v", err)
			return "", nil, err
		}
	default:
		err = RemoveResources(event)
		if err != nil {
			log.Printf("Error removing resources: %v", err)
			return "", nil, err
		}
	}

	log.Printf("Returning: %v, %#v", physicalResourceID, data)
	return physicalResourceID, data, nil
}

func GetInstancesInASG(ctx context.Context, asgName string) (instances []ec2Types.Instance, err error) {
	var cfg aws.Config
	cfg, err = config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Error loading AWS config: %v", err)
		return
	}

	instances, err = GetInstancesInASGwithConfig(ctx, asgName, cfg)

	return
}

// RemoveResources - Only needed if there were actual resources created.
func RemoveResources(event cfn.Event) error {
	log.Printf("RemoveResources(%#v)", event)
	log.Printf("PhysicalResourceId %v", event.PhysicalResourceID)
	// Implement resource removal logic if needed
	return nil
}

// compareSlices checks if two slices have the same members
func compareSlices(slice1, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	counts := make(map[string]int)

	for _, item := range slice1 {
		counts[item]++
	}

	for _, item := range slice2 {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}

	for _, count := range counts {
		if count != 0 {
			return false
		}
	}

	return true
}

// DescribeASGWithConfig - Describe the Auto Scaling group
func DescribeASGwithConfig(ctx context.Context, asgName string, cfg aws.Config) (autoScalingGroups []asgTypes.AutoScalingGroup, err error) {
	asgSvc := autoscaling.NewFromConfig(cfg)
	err, autoScalingGroups = DescribeASGwithClient(ctx, asgName, asgSvc)

	return
}

// DescribeASG - Describe the Auto Scaling group
func DescribeASG(ctx context.Context, asgName string) (autoScalingGroups []asgTypes.AutoScalingGroup, err error) {
	var cfg aws.Config
	cfg, err = config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("Error loading AWS config: %v", err)
		return
	}

	autoScalingGroups, err = DescribeASGwithConfig(ctx, asgName, cfg)

	return
}

func DescribeASGwithClient(ctx context.Context, asgName string, asgSvc autoscaling.DescribeAutoScalingGroupsAPIClient) (err error, autoScalingGroups []asgTypes.AutoScalingGroup) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{asgName},
	}

	result, err := asgSvc.DescribeAutoScalingGroups(ctx, input)
	if err != nil {
		return fmt.Errorf("Failed to describe Auto Scaling group: %w", err), nil
	}

	return nil, result.AutoScalingGroups
}

type InstanceInfo struct {
	ec2Instances []ec2Types.Instance
}

func GetInstancesInASGwithConfig(ctx context.Context, asgName string, cfg aws.Config) (instances []ec2Types.Instance, err error) {
	asgSvc := autoscaling.NewFromConfig(cfg)
	var asgInstances []asgTypes.Instance
	asgInstances, err = GetInstancesInASGwithClient(ctx, asgName, asgSvc)
	ec2Svc := ec2.NewFromConfig(cfg)

	// put the asgInstance IDs into a slice of strings
	instanceIds := make([]string, 0, len(asgInstances))
	for _, asgInstance := range asgInstances {
		if asgInstance.LifecycleState == asgTypes.LifecycleStatePending ||
			asgInstance.LifecycleState == asgTypes.LifecycleStatePendingWait ||
			asgInstance.LifecycleState == asgTypes.LifecycleStatePendingProceed ||
			asgInstance.LifecycleState == asgTypes.LifecycleStateInService {
			log.Printf("Instance: %#v", asgInstance)
			instanceIds = append(instanceIds, *asgInstance.InstanceId)
		}
	}

	// Get the asgInstance information and put into the InstanceInfo slice
	// Get the asgInstance information
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	}
	var nextToken *string
	var decribeInstanceOutput *ec2.DescribeInstancesOutput

	// Loop to make sure we get all the data
	for {
		describeInput.NextToken = nextToken
		decribeInstanceOutput, err = ec2Svc.DescribeInstances(ctx, describeInput)
		if err != nil {
			log.Printf("Error describing instances: %v", err)
			return
		}
		for _, reservation := range decribeInstanceOutput.Reservations {
			instances = append(instances, reservation.Instances...)
		}
		if decribeInstanceOutput.NextToken == nil {
			break
		}

		nextToken = decribeInstanceOutput.NextToken
	}

	return
}

func GetInstancesInASGwithClient(ctx context.Context, asgName string, asgSvc autoscaling.DescribeAutoScalingGroupsAPIClient) (instances []asgTypes.Instance, err error) {
	var autoscalingGroups []asgTypes.AutoScalingGroup
	err, autoscalingGroups = DescribeASGwithClient(ctx, asgName, asgSvc)
	if err != nil {
		return
	}
	for _, asg := range autoscalingGroups {
		instances = append(instances, asg.Instances...)
	}

	return
}
