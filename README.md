# Zoo-as-a-Service (Pulumi Demo)
The purpose of this repo, originally, was to teach myself how to do a few things:
- Write code using the Go programming language
- Use Pulumi to deploy cloud resources
- Deploy a simple API

# Before You Begin
You'll need to perform a few basic steps to configure your workspace in order to work with this repo.

> **Warning**
> Note that these instructions are from the point-of-view of an engineer using a :apple: macOS workspace. Instructions for engineers on Windows or *nix operating systems may differ.

## Clone This Repo

```bash
git clone git@github.com.au/aholden85/pulumi-demo-go.git
```

## Download/Install Homebrew
Refer to the instructions on [the Homebrew page](https://brew.sh/), or copy/paste the command below into your terminal:
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

## Download/Install Pulumi
Refer to the instructions on [the Pulumi "Get Started with Pulumi" page](https://www.pulumi.com/docs/get-started/install/), or copy/paste the command below into your terminal:
```bash
brew install pulumi/tap/pulumi
```

## Configure Your Backend
Pulumi supports two classes of state backends for storing your infrastructure state:

- **Service:** a managed cloud experience using the online or self-hosted Pulumi Cloud application
- **Self-Managed:** a manually managed object store, including AWS S3, Azure Blob Storage, Google Cloud Storage, any AWS S3 compatible server such as Minio or Ceph, or your local filesystem

Follow the instructions on [the Pulumi "State and Backends" page](https://www.pulumi.com/docs/intro/concepts/state/) for details on how to configure/log into a backend.

### [Service (Pulumi Cloud)](https://www.pulumi.com/docs/intro/concepts/state/#pulumi-cloud-backend)
You will need a Pulumi account, which you can create by following the instructions on [the Pulumi "Pulumi Cloud Accounts" page](https://www.pulumi.com/docs/intro/pulumi-cloud/accounts/).

Alternatively, if you would prefer to use a self-hosted deployment of Pulumi Cloud, follow the instructions on [the Pulumi "Self-Hosting the Pulumi Cloud" page](https://www.pulumi.com/docs/guides/self-hosted/).

### [Self-Managed](https://www.pulumi.com/docs/intro/concepts/state/#using-a-self-managed-backend)
To use a self-managed backend, specify a storage endpoint URL as `pulumi login`’s `<backend-url>` argument:
- `s3://<bucket-path>`
- `azblob://<container-path>`
- `gs://<bucket-path>`
- `file://<fs-path>`

This will tell Pulumi to store state in AWS S3, Azure Blob Storage, Google Cloud Storage, or the local filesystem, respectively.

## Set Your AWS Profile and Region
This project, specifically, is currently targeting AWS. To deploy it, you will need to configure your AWS region and profile.

You can configure these as environment variables, using the commands below:
```bash
export AWS_REGION=your-aws-region
export AWS_PROFILE=your-aws-profile
```

Alternatively, you can use `pulumi config set` to add AWS-specific variables, as shown below:
```bash
pulumi configure set aws:region your-aws-region
pulumi configure set aws:profile your-aws-profile
```

You can also edit the `iac/Pulumi.dev.yaml` file directly to include the following lines under the `config:` header:
```yaml
  aws:region: your-aws-region
  aws:profile: your-aws-profile
```

# Repo Structure
The repository contains two key folders:
- `assets`: this folder contains all of the animal facts and images, and the code for the Lambda functions that will service API requests hitting the API Gateway.
- `iac`: this folder contains all of the Pulumi-specific code for deploying, and testing, the AWS resouces.

## Folder Structure Diagram
```
	pulumi-demo-go
	├─  assets
	├─	iac
	└─  README.md
```

# IaC Configuration
You can specify what animal you want to deploy facts for by modifying the `iac/Pulumi.dev.yaml` file and updating the `animal:` value to your desired animal.

Currently supported animals are:
- `otter`
- `platypus`

You can add your own animal by creating a folder under the `assets/animals` folder. For specifics, refer to the [animals readme file](assets/animals/README.md).

# Deployed Infrastructure
This project will deploy the following resources into the target AWS account:
- `1x` DynamoDB Table
	- `Several` DynamoDB Table Items, depending on which animal you're deploying, and how many facts are in the `assets/animals/<animal>/facts.txt` file (each line is a fact)
- `1x` S3 Bucket and attached bucket policy to allow public access to the bucket and contained S3 objects
	- `Several` S3 Objects, depending on what animal you're deploying, and how many images are in the `assets/animals/<animal>/images` folder (each file, other than the `metadata.json` file is an image)
- `2x` Lambda Functions, one for the `Facts` endpoint, and one for the `Images` endpoint
- `2x` IAM Roles, one for each of the Lambda Functions, and attached IAM Role Policies to allow required permissions (get S3 Objects or DynamoDB Table items)
- `Several` API Gateway resources
	- `1x` API Gateway Deployment
	- `1x` API Gateway RestAPI
	- `1x` API Gateway Stage

# Utilisation of `Makefile`s
There are several `Makefile`s as part of this project:
- Each Lambda function has a `Makefile` that compiles the Go code and compresses it as a `.zip` file, ready to be deployed.
- The Pulumi code has a `Makefile` for `deploy`, `destroy` and `test` purposes.
  - Executing `make` from the `iac` folder will run the `integration` and `unit` tests before executing a `pulumi up` command.

# Testing
Using a combination of `go test` and Pulumi's testing framework, I have implemented **unit** and **integration** testing. **Property** testing is also possible, but has not been implemented at this stage.

## Integration Tests
Integration Tests deploy ephemeral infrastructure and run external tests against it. The implemented integration tests will ensure that the program compiles properly, and that the resource types and counts are correct.

The integration tests are included in the [integration_test.go](/iac/integration_test.go) file.

These tests can be executed using the following `Makefile` command:
```bash
make test-integration
```

Alternatively, they  can be executed using the following `go test` command:
```bash
go test -tags=integration
```

## Unit Tests
Unit Tests are fast in-memory tests that mock all external calls. The implemented unit tests will ensure that all resources have a lower-case name. 

The unit tests are included in the [unit_test.go](/iac/unit_test.go) file.

These tests can be executed using the following `Makefile` command:
```bash
make test-unit
```

Alternatively, they can be executed using the following `go test` command:
```bash
go test -tags=unit
```


# Deploying
If you've followed the instrutions above, you should be ready to deploy! To test and deploy the infrastructure, execute the `Makefile` from within the `iac` folder in the project:
```bash
make
```

Alternatively, you can execute the following `pulumi` command:
```bash
pulumi up
```

You should be presented with something similar to the following output:
```
Previewing update (dev)

View in Browser (Ctrl+O): <your preview URL will be here>

     Type                               Name                                   Plan       
 +   pulumi:pulumi:Stack                pulumi-demo-go-dev                     create     
 +   ├─ aws:dynamodb:Table              xaas-ddb-facts                         create     
 +   ├─ aws:dynamodb:TableItem          xaas-ddb-facts-X                       create     
 +   ├─ <!-- THERE WILL BE MULTIPLE dynamodb:TableItem HERE FOR EACH ANIMAL FACT -->
 +   ├─ aws:iam:Role                    xaas-lambda-facts-exec-role            create     
 +   ├─ aws:iam:Role                    xaas-lambda-images-exec-role           create     
 +   ├─ aws:iam:RolePolicy              xaas-lambda-facts-ddb-read-policy      create     
 +   ├─ aws:iam:RolePolicy              xaas-lambda-images-s3-read-policy      create     
 +   ├─ aws:iam:RolePolicyAttachment    xaas-lambda-facts-exec-role-cwpolicy   create     
 +   ├─ aws:iam:RolePolicyAttachment    xaas-lambda-images-exec-role-cwpolicy  create     
 +   ├─ aws:lambda:Function             xaas-lambda-facts                      create     
 +   ├─ aws:s3:Bucket                   xaas-s3-assets                         create     
 +   ├─ aws:s3:BucketObject             xaas-s3-assets-animalX.XXX             create     
 +   ├─ <!-- THERE WILL BE MULTIPLE s3:BucketObject HERE FOR EACH ANIMAL IMAGE -->   
 +   ├─ aws:s3:BucketPolicy             xaas-assets-policy                     create     
 +   ├─ aws:s3:BucketPublicAccessBlock  xaas-s3-assets-publicaccess-allow      create     
 +   ├─ aws:lambda:Function             xaas-lambda-images                     create     
 +   └─ aws-apigateway:index:RestAPI    xaas-apigw                             create     
 +      ├─ aws:apigateway:RestApi       xaas-apigw                             create     
 +      ├─ aws:apigateway:Deployment    xaas-apigw                             create     
 +      ├─ aws:lambda:Permission        xaas-apigw-2a7be15d                    create     
 +      ├─ aws:lambda:Permission        xaas-apigw-576bbc14                    create     
 +      └─ aws:apigateway:Stage         xaas-apigw                             create     


Outputs:
    url: output<string>

Resources:
    + XX to create

Do you want to perform this update?  [Use arrows to move, type to filter]
  yes
> no
  details
```

Select `yes` to deploy the infrastructure and take note of the output so you can test your API.

## Results

```
     Type                               Name                                   Status              
 +   pulumi:pulumi:Stack                pulumi-demo-go-dev                     created (67s)       
 +   ├─ aws:dynamodb:Table              xaas-ddb-facts                         created (10s)       
 +   ├─ aws:dynamodb:TableItem          xaas-ddb-facts-X                       created (Xs)        
 +   ├─ <!-- THERE WILL BE MULTIPLE dynamodb:TableItem HERE FOR EACH ANIMAL FACT -->
 +   ├─ aws:iam:Role                    xaas-lambda-facts-exec-role            created (2s)        
 +   ├─ aws:iam:Role                    xaas-lambda-images-exec-role           created (2s)        
 +   ├─ aws:iam:RolePolicy              xaas-lambda-facts-ddb-read-policy      created (1s)        
 +   ├─ aws:iam:RolePolicy              xaas-lambda-images-s3-read-policy      created (0.84s)     
 +   ├─ aws:iam:RolePolicyAttachment    xaas-lambda-facts-exec-role-cwpolicy   created (1s)        
 +   ├─ aws:iam:RolePolicyAttachment    xaas-lambda-images-exec-role-cwpolicy  created (1s)        
 +   ├─ aws:lambda:Function             xaas-lambda-images                     created (22s)       
 +   ├─ aws:s3:Bucket                   xaas-s3-assets                         created (5s)        
 +   ├─ aws:s3:BucketObject             xaas-s3-assets-animalX.XXX             created (xs)    
 +   ├─ <!-- THERE WILL BE MULTIPLE s3:BucketObject HERE FOR EACH ANIMAL IMAGE -->   
 +   ├─ aws:s3:BucketPolicy             xaas-assets-policy                     created (0.93s)     
 +   ├─ aws:s3:BucketPublicAccessBlock  xaas-s3-assets-publicaccess-allow      created (1s)        
 +   ├─ aws:lambda:Function             xaas-lambda-facts                      created (30s)       
 +   └─ aws-apigateway:index:RestAPI    xaas-apigw                             created (13s)       
 +      ├─ aws:apigateway:RestApi       xaas-apigw                             created (2s)        
 +      ├─ aws:apigateway:Deployment    xaas-apigw                             created (1s)        
 +      ├─ aws:lambda:Permission        xaas-apigw-576bbc14                    created (1s)        
 +      ├─ aws:lambda:Permission        xaas-apigw-2a7be15d                    created (1s)        
 +      └─ aws:apigateway:Stage         xaas-apigw                             created (0.97s)     


Outputs:
    url: <output_url>

Resources:
    + XX created

Duration: 1m12s
```

## Validation
You'll be able to query your APIs using the following endpoints:

### Facts

To retrieve a random fact, query `<output_url>/facts` with a `GET`

To retrieve a specific fact, query `<output_url>/facts?FactId=1` with a `GET`

### Images
To retrieve a random image, query, `<output_url>/images` with a `GET`