package amazon

import (
	"github.com/aws/aws-sdk-go/service/ec2"
)

func AvailabilityZones() ([]string, error) {
	answer := []string{}

	sess, _, err := NewAwsSession()
	if err != nil {
		return answer, err
	}

	svc := ec2.New(sess)
	input := &ec2.DescribeAvailabilityZonesInput{}

	result, err := svc.DescribeAvailabilityZones(input)
	if err != nil {
		return answer, err
	}
	for _, zone := range result.AvailabilityZones {
		if zone != nil && zone.ZoneName != nil {
			answer = append(answer, *zone.ZoneName)
		}
	}
	return answer, nil
}
