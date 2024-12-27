# ASGInfo

This is a CloudFormation Macro that will get information about whether an instance type is available in the AZs
where the instance could be built. Needed this because the `InstanceType` parameter was failing for some regions for the
`t4g.small` instance type in `us-east-1` region.

## Properties

| Property Name | Description                                                                                |
|---------------|--------------------------------------------------------------------------------------------|
| InstanceType  | The Instance Type we want to check for availability in all the subnets                     |  
| Subnets       | The subnets we want to check against (will be all in the VPC generally) passed as an array |

## Return Values

| Name                  | Description                                                 |
|-----------------------|-------------------------------------------------------------|
| AvailableInAZs        | The zones that have the instance type (array)               |
| AvailableInASubnetIds | The SubnetIds that have the instance type (comma separated) |

### CloudFormation snippet

```yaml
  # Custom resource to get subset of zones that have the instance type available
  ASGInfo:
    Type: Custom::CheckInstanceTypeAvailability
    Properties:
      ServiceToken: arn:aws:lambda:${AWS::Region}:${AWS::AccountId}:function:ASGInfo
      InstanceType: !Ref InstanceType
      Subnets:
        Fn::Split:
          - ","
          - Fn::ImportValue: !Sub '${VPCDataStackName}-SubnetIds' 
```

## ASGInfo.go

This is the Lambda code for the Macro. It gets the VPC ID and returns the subnet IDs and count of subnets in the VPC.

## create.sh

This script creates the role with the permissions that the Lambda needs and deploys the Lambda initially. Probably could
combine this with the `update.sh` to make things easier.

To run this script, you must have a valid session, and the region must be set. In most cases this simply means you set
these variables:

- `AWS_PROFILE` the name of a profile in your `~/.aws/config` or `~/.aws/credentials` file, e.g. - `wf-bia-dev`
- `AWS_REGION` the region you are going to deploy in (normally this will be `us-west-2`)

The steps of this script are as follows:

1. creates a role with the `trust-policy.json` and then attaches policies that the Lambda will need.
1. builds the `ASGInfo.go`
1. Deploys the Lambda to the proper account.

## update.sh

This script updates the Lambda code (particularly useful during development) same variables apply as for `create.sh`

The steps of this script are as follows:

1. builds the `ASGInfo.go`
1. Deploys the Lambda to the proper account.

## trust-policy.json

The Trust policy for Lambda to use the role that is created in `create.sh`