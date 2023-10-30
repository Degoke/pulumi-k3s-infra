package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	// "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ebs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	// "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	// corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	// metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	// "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"


	"os"
	"fmt"
	// "encoding/json"
	// "errors"
	"gopkg.in/yaml.v2"
// 	"crypto/rsa"
// 	"crypto/rand"
//   "encoding/pem" 
)

type KubeConfig struct {
	APIVersion     string `yaml:"apiVersion"`
	Clusters       []struct {
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
			Server                   string `yaml:"server"`
		} `yaml:"cluster"`
		Name string `yaml:"name"`
	} `yaml:"clusters"`
	Contexts       []struct {
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
		Name string `yaml:"name"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
	Kind           string `yaml:"kind"`
	Preferences    map[string]interface{} `yaml:"preferences"`
	Users          []struct {
		Name string `yaml:"name"`
		User struct {
			ClientCertificateData string `yaml:"client-certificate-data"`
			ClientKeyData         string `yaml:"client-key-data"`
		} `yaml:"user"`
	} `yaml:"users"`
}


func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		// A path to the EC2 keypair's public key:
		publicKeyPath := cfg.Require("publicKeyPath")
		// A path to the EC2 keypair's private key:
		privateKeyPath := cfg.Require("privateKeyPath")

		fmt.Printf(publicKeyPath)
		fmt.Printf(privateKeyPath)

		ec2InstanceSize := cfg.Get("ec2InstanceSize")
		if ec2InstanceSize == "" {
        	ec2InstanceSize = "t2.medium"
		}

		ec2AmiId := cfg.Get("ec2AmiId")
		if ec2AmiId == "" {
        	ec2AmiId = "ami-0d52744d6551d851e"
		}


		// Read in the public key for easy use below.
		publicKeyBytes, err := os.ReadFile(publicKeyPath)
		if err != nil {
			return err
		}
		publicKey := pulumi.String(string(publicKeyBytes))
		// Read in the private key for easy use below (and to ensure it's marked a secret!)
		privateKeyBytes, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return err
		}
		privateKey := pulumi.String(string(privateKeyBytes))


		// Create an AWS resource (S3 Bucket)
		bucket, err := s3.NewBucket(ctx, "DevBucket", nil)
		if err != nil {
			return err
		}

		// Create Security Group
		securityGroup, err := ec2.NewSecurityGroup(ctx, "DevSg", &ec2.SecurityGroupArgs{
			Description: pulumi.String("Allow inbound and outbound traffic"),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					FromPort:   pulumi.Int(2379),
					ToPort:     pulumi.Int(2380),
					Protocol:   pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort:   pulumi.Int(6443),
					ToPort:     pulumi.Int(6443),
					Protocol:   pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort:   pulumi.Int(10250),
					ToPort:     pulumi.Int(10250),
					Protocol:   pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					Protocol:   pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort:   pulumi.Int(8080),
					ToPort:     pulumi.Int(8080),
					Protocol:   pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort:   pulumi.Int(443),
					ToPort:     pulumi.Int(443),
					Protocol:   pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
            Egress: ec2.SecurityGroupEgressArray{
                &ec2.SecurityGroupEgressArgs{
                    FromPort:   pulumi.Int(0),
                    ToPort:     pulumi.Int(0),
                    Protocol:   pulumi.String("-1"), // This represents all protocols.
                    CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")}, // This represents all IP addresses.
                },
            },
		})
		if err != nil {
			return err
		}

		// Create a keypair to access the EC2 instance:
		devKeypair, err := ec2.NewKeyPair(ctx, "DevKeypair", &ec2.KeyPairArgs{
			PublicKey: pulumi.String(string(publicKey)),
		})
		if err != nil {
			return err
		}


		// Create the user data script for the EC2 instance that will form the K8s cluster
		// script := `#!/bin/bash
		// # install dependencies for and k38s
		// sudo apt-get update
		// curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644
		// `

		instance, err := ec2.NewInstance(ctx, "DevK3s", &ec2.InstanceArgs{
			InstanceType:        pulumi.String(ec2InstanceSize),
			VpcSecurityGroupIds: pulumi.StringArray{securityGroup.ID()},
			// IamInstanceProfile:  pulumi.String(role.Name),
			Ami:                 pulumi.String(ec2AmiId),
			// UserData:            pulumi.String(script),
			KeyName:      devKeypair.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("DevK3s"),
			},
		})

		if err != nil {
			return err
		}

		// // Create a new AWS EBS Volume
		// volume, err := ebs.NewVolume(ctx, "extraStorage", &ebs.VolumeArgs{
		// 	AvailabilityZone: instance.AvailabilityZone,  // Same as the EC2 instance
		// 	Size:             pulumi.Int(8),             // 8 GiB
		// 	Tags:             pulumi.StringMap{"Name": pulumi.String("extraStorage")},
		// })

		// if err != nil {
		// 	return err
		// }

		// // Attach the EBS Volume to the EC2 instance
		// _, err = ec2.NewVolumeAttachment(ctx, "extraStorageAttachment", &ec2.VolumeAttachmentArgs{
		// 	DeviceName: pulumi.String("/dev/sdh"),  // Linux device name on the EC2 instance
		// 	InstanceId: instance.ID(),
		// 	VolumeId:   volume.ID(),
		// })

		// if err != nil {
		// 	return err
		// }

		var server string
		var ip string
		instance.PublicIp.ApplyT(func (args string) error {
			server = fmt.Sprintf("https://%s:6443", args)
			ip = args
			return nil
		})

		installK3s, err := remote.NewCommand(ctx, "installK3s", &remote.CommandArgs{
			Connection: &remote.ConnectionArgs{
				Host:       instance.PublicIp,
				Port:       pulumi.Float64(22),
				User:       pulumi.String("ubuntu"),
				PrivateKey: privateKey,
			},
			Create:
			  // First, we use `until` to monitor for the k3s.yaml (our kubeconfig) being created.
			  // Then we sleep 10, just in-case the k3s server needs a moment to become healthy. Sorry?
			  pulumi.String(fmt.Sprintf("curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC=--tls-san %s sh -s - --write-kubeconfig-mode 644", ip)),
		  });

		  if err != nil {
			return err
		}

		fetchKubeconfig, err := remote.NewCommand(ctx, "fetch-kubeconfig", &remote.CommandArgs{
			Connection: &remote.ConnectionArgs{
				Host:       instance.PublicIp,
				Port:       pulumi.Float64(22),
				User:       pulumi.String("ubuntu"),
				PrivateKey: privateKey,
			},
			Create:
			  // First, we use `until` to monitor for the k3s.yaml (our kubeconfig) being created.
			  // Then we sleep 10, just in-case the k3s server needs a moment to become healthy. Sorry?
			  pulumi.String("until [ -f /etc/rancher/k3s/k3s.yaml ]; do sleep 5; done; cat /etc/rancher/k3s/k3s.yaml; sleep 10;"),
		  }, pulumi.DependsOn([]pulumi.Resource{installK3s}));

		  if err != nil {
			return err
		}

		fmt.Println(server)
		fmt.Printf("%v", fetchKubeconfig.Stdout)

		// // Unmarshal the YAML configuration
		// var kubeConfig KubeConfig
		// err = yaml.Unmarshal([]byte(string(fetchKubeconfig.Stdout)), &kubeConfig)
		// if err != nil {
		// return err
		// }

		jsonConfig := fetchKubeconfig.Stdout.ApplyT(
			func(v string) (pulumi.StringOutput, error) {
				// fmt.Printf("%v", v)
				// var jsonV any
				var kubeConfig KubeConfig
				err = yaml.Unmarshal([]byte(v), &kubeConfig)
				if err != nil {
					return pulumi.StringOutput{}, err
				}
				// fmt.Printf("%v", kubeConfig)
				address := fmt.Sprintf("https://%s:6443", "13.230.34.83")
				// Update the server field
				kubeConfig.Clusters[0].Cluster.Server = address

				// Marshal the updated configuration back to YAML
				updatedYAML, err := yaml.Marshal(&kubeConfig)

				// fmt.Printf("%v", string(updatedYAML))
				if err != nil {
				return pulumi.StringOutput{}, err
				}

				return pulumi.String(string(updatedYAML)).ToStringOutput(), nil
			}).(pulumi.StringOutput)

		_, err = kubernetes.NewProvider(ctx, "Devk3s", &kubernetes.ProviderArgs{
			Kubeconfig: fetchKubeconfig.Stdout.ApplyT(
				func(v string) (pulumi.StringOutput, error) {
					// var jsonV any
					fmt.Printf("%v", v)
					var kubeConfig KubeConfig
					err = yaml.Unmarshal([]byte(v), &kubeConfig)
					if err != nil {
						return pulumi.StringOutput{}, err
					}
					// fmt.Printf("%v", kubeConfig)
					// address := fmt.Sprintf("https://%s:6443", "13.230.34.83")
					// Update the server field
					
					kubeConfig.Clusters[0].Cluster.Server = server
	
					// Marshal the updated configuration back to YAML
					updatedYAML, err := yaml.Marshal(&kubeConfig)
	
					// fmt.Printf("%v", string(updatedYAML))
					if err != nil {
					return pulumi.StringOutput{}, err
					}
	
					return pulumi.String(string(updatedYAML)).ToStringOutput(), nil
				}).(pulumi.StringOutput),
		})

		// _, err = helm.NewRelease(ctx, "nginx-ingress", &helm.ReleaseArgs{
		// 	Chart:   pulumi.String("ingress-nginx"),
		// 	Version: pulumi.String("4.8.1"),
		// 	RepositoryOpts: helm.RepositoryOptsArgs{
		// 		Repo: pulumi.String("https://kubernetes.github.io/ingress-nginx"),
		// 	},
		// 	// SkipCrds: pulumi.Bool(true),
		// 	// Values: pulumi.Map{
		// 	// 	"controller": pulumi.Map{
		// 	// 		"enableCustomResources": pulumi.Bool(false),
		// 	// 		"appprotect" : pulumi.Map{
		// 	// 			"enable": pulumi.Bool(false),
		// 	// 		},
		// 	// 		"appprotectddos" : pulumi.Map{
		// 	// 			"enable": pulumi.Bool(false),
		// 	// 		},
		// 	// 	},
		// 	// },
		// }, pulumi.Provider(provider))
		// if err != nil {
		// 	return err
		// }

		// _, err = v1.NewDeployment(ctx, "my-k8s-deployment", &v1.DeploymentArgs{
		// 	Metadata: &metav1.ObjectMetaArgs{
		// 		Name: pulumi.StringPtr("my-deployment"),
		// 	},
		// 	Spec: &v1.DeploymentSpecArgs{
		// 		Replicas: pulumi.Int(3),
		// 		Selector: &metav1.LabelSelectorArgs{
		// 			MatchLabels: pulumi.StringMap{
		// 				"app": pulumi.String("my-app"),
		// 			},
		// 		},
		// 		Template: &corev1.PodTemplateSpecArgs{
		// 			Metadata: &metav1.ObjectMetaArgs{
		// 				Labels: pulumi.StringMap{
		// 					"app": pulumi.String("my-app"),
		// 				},
		// 			},
		// 			Spec: &corev1.PodSpecArgs{
		// 				Containers: corev1.ContainerArray{
		// 					&corev1.ContainerArgs{
		// 						Name:  pulumi.String("my-app"),
		// 						Image: pulumi.String("nginx:1.14.2"),
		// 						Ports: corev1.ContainerPortArray{
		// 							&corev1.ContainerPortArgs{
		// 								ContainerPort: pulumi.Int(80),
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// }, pulumi.Provider(provider))
		// if err != nil {
		// 	return err
		// }

		// //copy kube folder
		// _, err = remote.NewCopyFile(ctx, "copyKubeCmd", &remote.CopyFileArgs{
		// 	Connection: &remote.ConnectionArgs{
		// 		Host:       instance.PublicIp,
		// 		Port:       pulumi.Float64(22),
		// 		User:       pulumi.String("ubuntu"),
		// 		PrivateKey: privateKey,
		// 	},
		// 	LocalPath: pulumi.String("kube"),
		// 	RemotePath: pulumi.String("/home/ubuntu/"),
		// })
		
		// if err != nil {
		// 	return err
		// }

		// // Run a script to i stall nginx-ingress remote machine.
		// updatePythonCmd, err := remote.NewCommand(ctx, "updatePythonCmd", &remote.CommandArgs{
		// 	Connection: &remote.ConnectionArgs{
		// 		Host:       wordpressEip.PublicIp,
		// 		Port:       pulumi.Float64(22),
		// 		User:       pulumi.String("ec2-user"),
		// 		PrivateKey: privateKey,
		// 	},
		// 	Create: pulumi.String("(sudo yum update -y || true);" +
		// 		"(sudo yum install python35 -y);" +
		// 		"(sudo yum install amazon-linux-extras -y)\n"),
		// })
		// if err != nil {
		// 	return err
		// }

		ctx.Export("bucketName", bucket.ID())
		ctx.Export("Worker public IP", instance.PublicIp)
		ctx.Export("config", fetchKubeconfig.Stdout)
		ctx.Export("pro", jsonConfig)
		return nil
	})
}
