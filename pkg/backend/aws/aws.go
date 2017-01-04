package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"strings"
)

// // Attacher manages attachment and mounting of a volume.
// type VolumeAttacher struct{}

// // Detacher manages detachment and unmounting of a volume.
// type VolumeDetacher struct{}

// // Manager TODO
// type Manager struct {
// 	Attacher
// 	Detacher
// }

// // New TODO
// func New() *Manager {
// 	return &Manager{}
// }

// // Attach TODO
// func (Attacher) Attach() {
// 	// TODO
// 	fmt.Println("Attaching volume.")
// }

// // WaitForAttach TODO
// func (Attacher) WaitForAttach() {
// 	// TODO
// 	fmt.Println("Waiting for volume attachment.")
// }

// // Detach TODO
// func (Detacher) Detach() {
// 	// TODO
// 	fmt.Println("Dettaching volume.")
// }

// Backend TODO
type Backend struct {
	Config
	NodeID           string
	ec2c             *ec2.EC2
	instanceID       string
	vpcID            string
	az               string
	region           string
	volume           *volume
	networkInterface *networkInterface
}

// Config TODO
type Config struct {
	BlockDeviceName string
	Filter          string
}

// New TODO
func New(cfg Config) *Backend {
	return &Backend{
		Config: cfg,
	}
}

func (i *Backend) getMetadata() error {
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
func buildFilters(i Backend) []*ec2.Filter {
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
	if i.Filter != "" {
		kvs := strings.Split(i.Filter, ",")
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
	NodeID       string
	attachmentID string
	IPAddress    string
}

func findNetworkInterfaces(i *Backend, ec2c *ec2.EC2, f []*ec2.Filter) ([]networkInterface, error) {
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
		n.NodeID = getResourceTagValue(*i.NetworkInterfaceId, "NodeID", ec2c)
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
	NodeID     string
	attachedTo string
}

func findVolumes(i *Backend, ec2c *ec2.EC2, f []*ec2.Filter) ([]volume, error) {
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
		v.NodeID = getResourceTagValue(*i.VolumeId, "NodeID", ec2c)
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
func (i *Backend) attachVolume(v volume, ec2c *ec2.EC2) error {
	params := &ec2.AttachVolumeInput{
		Device:     aws.String(i.BlockDeviceName),
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
func (i *Backend) attachNetworkInterface(n networkInterface, ec2c *ec2.EC2) error {
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
func (i *Backend) dettachNetworkInterface() error {
	log.Printf("Detaching network interface: %q.\n", i.networkInterface.id)
	_, err := i.ec2c.DetachNetworkInterface(&ec2.DetachNetworkInterfaceInput{
		AttachmentId: &i.networkInterface.attachmentID,
	})
	if err != nil {
		log.Printf("Failed to dettach network interface %q: %q.\n", i.networkInterface.id, err)
		return err
	}
	i.networkInterface = nil
	return nil
}

// disableSourceDestCheck sets SourceDestCheck attribute to false on all
// instance network interfaces.
func disableSourceDestCheck(instanceID string, ec2c *ec2.EC2) error {
	i, err := ec2c.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)}},
	)
	if err != nil {
		return err
	}
	for _, n := range i.Reservations[0].Instances[0].NetworkInterfaces {
		attr := &ec2.ModifyNetworkInterfaceAttributeInput{
			NetworkInterfaceId: n.NetworkInterfaceId,
			SourceDestCheck:    &ec2.AttributeBooleanValue{Value: aws.Bool(false)},
		}
		log.Printf("Disabling SourceDestCheck on %q network interface.\n", *n.NetworkInterfaceId)
		_, err := ec2c.ModifyNetworkInterfaceAttribute(attr)
		if err != nil {
			log.Printf("Failed to disable SourceDestCheck attribute of %q network interface: %q.\n", n.NetworkInterfaceId, err)
		}
	}
	return nil
}

// =================================================================
func main() {
	flag.Parse()

	if flag.NArg() > 0 || opts.help {
		fmt.Fprintf(os.Stderr, "Usage: %q [OPTION]...\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	if opts.version {
		fmt.Fprintln(os.Stderr, Version)
		os.Exit(0)
	}

	var i instance
	err := i.getMetadata()
	if err != nil {
		log.Fatalf("Issues getting instance metadata properties. Exiting..")
	}
	ec2c = ec2.New(session.New(), aws.NewConfig().WithRegion(i.region))
	disableSourceDestCheck(i.id, ec2c)
	filters = buildFilters(i)

	for {
		run(&i)
		time.Sleep(120 * time.Second)
	}
}

func run(i *instance) {
	// Iterate over found volumes and check if one of them is attached to the
	// instance, then update i.volume accordingly.
	volumes, err := findVolumes(i, ec2c, filters)
	if err != nil {
		log.Println(err)
	} else {
		for _, v := range volumes {
			if i.volume == nil && v.attachedTo == i.id && !v.available {
				log.Printf("Found attached volume: %q.\n", v.id)
				i.volume = &v
				break
			}
			if i.volume != nil && i.volume.id == v.id && v.available {
				i.volume = nil
				break
			}
		}
	}

	// Iterate over found network interfaces and see if one of them is attached
	// to the instance, then update i.networkInterface accordingly.
	networkInterfaces, err := findNetworkInterfaces(i, ec2c, filters)
	if err != nil {
		log.Println(err)
	} else {
		for _, n := range networkInterfaces {
			if i.networkInterface == nil && n.attachedTo == i.id && !n.available {
				log.Printf("Found attached network interface: %q.\n", n.id)
				i.networkInterface = &n
				break
			}
			if i.networkInterface != nil && i.networkInterface.id == n.id && n.available {
				i.networkInterface = nil
				break
			}
		}
	}

	// If nothing is attached, then pick an available volume. We never want to
	// attach a network interface if there is no volume attached first.
	if i.volume == nil && i.networkInterface == nil {
		log.Println("Neither a volume, nor a network interface are attached.")
		for _, v := range volumes {
			if v.available {
				i.attachVolume(v, ec2c)
				break
			}
		}
		if i.volume == nil {
			log.Println("No available volumes found.")
		}
		if i.volume != nil {
			for _, n := range networkInterfaces {
				if n.available && i.volume.nodeID == n.nodeID {
					_ = i.attachNetworkInterface(n, ec2c)
					break
				}
				log.Println("No available network interfaces found.")
			}
		} else {
			log.Println("No volumes appear to be attached, skipping network interface attachment.")
		}
	}

	// If volume is attached, but network interface is not, then find a
	// matching available network interface and attach it.
	if i.volume != nil && i.networkInterface == nil {
		for _, n := range networkInterfaces {
			if n.available && n.nodeID == i.volume.nodeID {
				_ = i.attachNetworkInterface(n, ec2c)
				break
			}
		}
	}

	// If network interface is attached, but volume is not, then find a
	// matching available volume and attach it. If we cannot find a matching
	// volume after 3 tries, we release the network interface.
	if i.networkInterface != nil && i.volume == nil {
		if volumeAttachTries > 2 {
			log.Println("Unable to attach a matching volume after 3 retries.")
			if err := i.dettachNetworkInterface(); err == nil {
				volumeAttachTries = 0
			}
		}
		for _, v := range volumes {
			if v.available && v.nodeID == i.networkInterface.nodeID {
				log.Printf("Found a matching volume %q with NodeID %q.\n", v.id, v.nodeID)
				if err := i.attachVolume(v, ec2c); err == nil {
					volumeAttachTries = 0
					break
				}
			}
		}
		if i.volume == nil {
			volumeAttachTries++
		}
	}

	// FIXME: below could be cleaned up with less if statements maybe
	// Set node ID. If specified, create and mount the file system.
	if i.volume != nil && i.networkInterface != nil {
		if i.volume.nodeID == i.networkInterface.nodeID {
			if i.nodeID != i.volume.nodeID {
				i.nodeID = i.volume.nodeID
				log.Printf("Node ID is %q.\n", i.nodeID)
				writeEnvFile(opts.envFile, *i)
			}
		}
		// Set nodeID only when both volume and network interface are attached and their node IDs match.
		if i.volume.nodeID != i.networkInterface.nodeID {
			log.Printf("Something has gone wrong, volume and network interface node IDs do not match.")
		}
		if opts.createFs {
			if !hasFs(opts.blockDevice, opts.fsType) {
				mkfs(opts.blockDevice, opts.fsType)
			}
		}
		if opts.mountFs {
			if hasFs(opts.blockDevice, opts.fsType) && !isMounted(opts.blockDevice) {
				mount(opts.blockDevice, opts.mountPoint, opts.fsType)
			}
		}
	}
}
