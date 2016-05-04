## smilodon
Smilodon manages attachment of EBS and ENI pairs in AWS EC2 auto scaling groups.

Think about zookeeper, etcd or similar datastores, where a data volume and IP
address have to always go together. Achieving that can be tricky, especially
if you want to take advantage of EC2 auto scaling groups.


### Getting Started
* First you need to create a number of EBS and ENI resources and tag them with
  `NodeID`. `NodeID` tag value can be anything as long as you have a matching
  EBS and ENI pair with the same `NodeID` tag value.

* Secondly, create an autoscaling group with a number of instances with
  smilodon provisioned. You can have more instances in auto scaling group than
  you have EBS+ENI pairs.

When smilodon starts it will try to find a matching EBS+ENI pair and attach
them. EBS gets attached first and ENI second. You can tell smilodon to create a
file system and mount it for convenience.

Smilodon does not know how to start a service, but that work is in progress
right now.

If your distro uses systemd, then you can easily tell your service unit to
watch for when a specific mount point is ready and start the service unit.


### Required AWS Permissions
You have two options here.
- Create an IAM user and provide smilodon with AWS API credentials via
  environment variables.

- Define an IAM instance role and reference that in the LaunchConfiguration.

I recommend the latter, but regardless which option you pick, you need to allow
the following IAM permissions (below could be a lot more granular and specific):

```yaml
Resource: "*"
Action:
  - ec2:DescribeInstances
  - ec2:DescribeTags
  - ec2:DescribeNetworkInterfaces
  - ec2:DescribeVolumes
  - ec2:AttachVolume
  - ec2:AttachNetworkInterface
  - ec2:DetachNetworkInterface
```


### Configuration
Configuration is done using command line flags - `smilodon --help`.


### Filtering AWS Resources
It is very likely that you have many EBS volumes and ENI devices in your AWS
account.

When you create your EBS+ENI resources, it makes sense to tag them with a
service name or something sensible, so that smilodon attaches correct resources.

Only `NodeID` tag is required and the rest can be arbitrary. For example, you
could tag your resources with the following tags:

```
NodeID=[0-5]
Env=development
Service=etcd
Project=<any value>
```

To tell smilodon to only look for resources with the above tags, do this:
```
smilodon --filters='Env=development,Service=etcd,tag-key=Project'
```

As you can see above, last filter matches on any value of tag `Project`. You
can also filter on a bunch of other AWS specific filters.

