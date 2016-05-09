package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"strings"
)

type instance struct {
	id               string
	nodeID           string
	vpc              string
	az               string
	region           string
	volume           *volume
	networkInterface *networkInterface
}

func (i *instance) getMetadata() error {
	// Get instance id
	metadata := ec2metadata.New(session.New())
	id, err := metadata.GetMetadata("instance-id")
	if err != nil {
		log.Printf("Failed to get instance ID from the metadata service: %q.\n", err)
		return err
	}
	i.id = id

	// Get instance region
	region, err := metadata.Region()
	if err != nil {
		log.Printf("Failed to get instance region from the metadata service: %q.\n", err)
		return err
	}
	i.region = region

	// Get AZ
	az, err := metadata.GetMetadata("placement/availability-zone")
	if err != nil {
		log.Printf("Failed to get instance AZ from the metadata service: %q.\n.", err)
		return err
	}
	i.az = az

	ec2c := ec2.New(session.New(), aws.NewConfig().WithRegion(i.region))
	// Get VpcId
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	}
	instances, err := ec2c.DescribeInstances(params)
	if err != nil {
		log.Printf("Failed to get instance VPC ID: %q.\n", err)
		return err
	}
	i.vpc = *instances.Reservations[0].Instances[0].VpcId
	return nil
}

func getResourceTagValue(id, tag string, ec2c *ec2.EC2) string {
	params := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(id),
				},
			},
			{
				Name: aws.String("key"),
				Values: []*string{
					aws.String(tag),
				},
			},
		},
	}
	resp, err := ec2c.DescribeTags(params)
	if err != nil {
		log.Printf("Cannot get tag %q of %q resource: %q.\n", tag, id, err)
		return ""
	}
	if len(resp.Tags) > 0 {
		for _, t := range resp.Tags {
			return *t.Value
		}
	}
	log.Printf("Cannot get tag %q of %q resource.\n", tag, id)
	return ""
}

// buildFilters builds a list of filters of type []*ec2.Filter. It parses
// optional filters via cli arguments.
func buildFilters(i instance) []*ec2.Filter {
	filters := []*ec2.Filter{
		{
			Name: aws.String("tag-key"),
			Values: []*string{
				aws.String("NodeID"),
			},
		},
		{
			Name: aws.String("availability-zone"),
			Values: []*string{
				aws.String(i.az),
			},
		},
	}
	if opts.filters != "" {
		kvs := strings.Split(opts.filters, ",")
		for _, i := range kvs {
			parts := strings.Split(i, "=")
			if len(parts) != 2 {
				continue
			}
			filter := &ec2.Filter{
				Name: aws.String(parts[0]),
				Values: []*string{
					aws.String(parts[1]),
				},
			}

			filters = append(filters, filter)
		}
	}
	return filters
}

type networkInterface struct {
	id           string
	available    bool
	attachedTo   string
	nodeID       string
	attachmentID string
	IPAddress    string
}

func findNetworkInterfaces(i *instance, ec2c *ec2.EC2, f []*ec2.Filter) ([]networkInterface, error) {
	vpcFilter := &ec2.Filter{
		Name: aws.String("vpc-id"),
		Values: []*string{
			aws.String(i.vpc),
		},
	}
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: append(f, vpcFilter),
	}
	r, err := ec2c.DescribeNetworkInterfaces(params)
	var ns []networkInterface
	if err != nil {
		log.Printf("Failed to find network interfaces: %q.\n", err)
		return ns, err
	}
	for _, i := range r.NetworkInterfaces {
		var n networkInterface
		n.id = *i.NetworkInterfaceId
		n.nodeID = getResourceTagValue(*i.NetworkInterfaceId, "NodeID", ec2c)
		n.IPAddress = *i.PrivateIpAddress
		if i.Attachment != nil {
			n.attachmentID = *i.Attachment.AttachmentId
		}
		if *i.Status == ec2.NetworkInterfaceStatusAvailable {
			n.available = true
		} else {
			n.available = false
			n.attachedTo = *i.Attachment.InstanceId
		}
		ns = append(ns, n)
	}
	return ns, nil
}

type volume struct {
	id         string
	available  bool
	nodeID     string
	attachedTo string
}

func findVolumes(i *instance, ec2c *ec2.EC2, f []*ec2.Filter) ([]volume, error) {
	params := &ec2.DescribeVolumesInput{
		Filters: f,
	}
	r, err := ec2c.DescribeVolumes(params)
	var vs []volume
	if err != nil {
		log.Printf("Failed to find volumes: %q.\n", err)
		return vs, err
	}
	for _, i := range r.Volumes {
		var v volume
		v.id = *i.VolumeId
		v.nodeID = getResourceTagValue(*i.VolumeId, "NodeID", ec2c)
		if *i.State == ec2.VolumeStateAvailable {
			v.available = true
		} else {
			for _, a := range i.Attachments {
				v.attachedTo = *a.InstanceId
			}
			v.available = false
		}
		vs = append(vs, v)
	}
	return vs, nil
}

// attachVolume attaches a volume v to an instance i.
func (i *instance) attachVolume(v volume, ec2c *ec2.EC2) error {
	params := &ec2.AttachVolumeInput{
		Device:     aws.String(opts.blockDevice),
		InstanceId: aws.String(i.id),
		VolumeId:   aws.String(v.id),
	}
	log.Printf("Attaching volume: %q.\n", v.id)
	// FIXME: wait for the attachment to happen?
	_, err := ec2c.AttachVolume(params)
	if err != nil {
		log.Printf("Failed to attach volume %q: %q.\n", v.id, err)
		return err
	}
	i.volume = &v
	return nil
}

// attachNetworkInterface attaches a network interface n to an instance i.
func (i *instance) attachNetworkInterface(n networkInterface, ec2c *ec2.EC2) error {
	params := &ec2.AttachNetworkInterfaceInput{
		InstanceId:         aws.String(i.id),
		NetworkInterfaceId: aws.String(n.id),
		DeviceIndex:        aws.Int64(1),
	}
	log.Printf("Attaching network interface: %q.\n", n.id)
	// FIXME: wait for the attachment to happen?
	_, err := ec2c.AttachNetworkInterface(params)
	if err != nil {
		log.Printf("Failed to attach network interface %q: %q.\n", n.id, err)
		return err
	}
	i.networkInterface = &n
	return nil
}

// dettachNetworkInterface detaches a network interface n.
func (i *instance) dettachNetworkInterface() error {
	log.Printf("Detaching network interface: %q.\n", i.networkInterface.id)
	_, err := ec2c.DetachNetworkInterface(&ec2.DetachNetworkInterfaceInput{
		AttachmentId: &i.networkInterface.attachmentID,
	})
	if err != nil {
		log.Printf("Failed to dettach network interface %q: %q.\n", i.networkInterface.id, err)
		return err
	}
	i.networkInterface = nil
	return nil
}
