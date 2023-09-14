# Registry

A simple Golang AWS Lambda and supporting infrastructure definitions to act as a simple, stateless registry that sits on top of github repositories.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Registering public keys](#registering-public-keys)
  - [Adding a public key](#adding-a-public-key)
  - [Removing a public key](#removing-a-public-key)
- [Contributing to the project](#contributing-to-the-project)
  - [Requirements](#requirements)
  - [Setup](#setup)
  - [Terraform Variables Configuration](#terraform-variables-configuration)
  - [Deployment](#deployment)
  - [DNS Configuration](#dns-configuration)
  - [API Routes and Curl Usage](#api-routes-and-curl-usage)
- [License](#license)

## Registering public keys

This section describes how to register public keys for the providers and is intended for authors of providers who want the users of their providers to be able to verify the authenticity of the provider binaries.

### Adding a public key

All keys are stored in the `lambda/internal/provider/keys` directory. That directory contains subdirectories, each of which is named after the GitHub namespace (username or organization name) that hosts one or more providers.

Inside that directory are one or more ASCII-armored public key files. The names of the files are not relevant to the registry code, but it is recommended that they have a `.asc` extension. It may also be a good idea to name the files using the registration date to make it easier for the reader to determine which key is the most recent. The contents of the file should be the ASCII-armored public key for the namespace.

When the user requests any provider in a given namespace, the registry will return all the registered public keys for that namespace. The user can then use these keys to verify the signature of the provider binary.

### Removing a public key

It is possible to remove a public key from the registry. To do so, simply delete the corresponding file from the `lambda/internal/provider/keys` directory. The next time the registry is deployed, the key will no longer be available.

This will however have an impact on the users of the provider, which will no longer be able to verify the authenticity of the provider binaries. In case of a leak it is thus recommended to re-sign all the provider binaries with a new key, and to register the new key in the registry.

## Contributing to the project

This section describes how to contribute to the project and is intended for developers who wish to contribute new features, bug fixes, or improvements to the project. It includes information about the requirements, setup, deployment, and DNS configuration.

### Requirements

- **Go**: The AWS Lambda function is written in Go. Ensure you have Go installed.
- **Terraform**: This project uses Terraform for infrastructure management. Make sure to have Terraform installed.
- **AWS CLI**: Ensure that the AWS CLI is installed and configured with the necessary permissions.

### Setup

1. **Clone the Repository**:

    ```bash
    git clone <repository_url>
    cd <repository_name>
    ```

2. **Set Up Go**:
   Navigate to the `lambda` directory and download the required Go modules.

    ```bash
    cd lambda
    go mod download
    ```

3. **Initialize Terraform**:
    From the root of the project:

    ```bash
    terraform init
    ```

### Terraform Variables Configuration

Before deploying the infrastructure, ensure you've set the required Terraform variables:

- **`github_api_token`**: Personal Access Token (PAT) from GitHub, required for interactions with the GitHub API. [Create a GitHub PAT](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) if you don't have one, it should have `public_repo, read:packages` access

- **`route53_zone_id`**: The ID of the Route 53 hosted zone, e.g., "Z008B5091482A026MN9AUQ"

- **`domain_name`**: The domain name you wish to manage. This should match or be a subdomain of the `route53_zone_name`.

To provide values for these variables:

- Use the `-var` flag during `terraform apply`, e.g., `terraform apply -var="github_api_token=YOUR_TOKEN"`.
- Or, populate a `terraform.tfvars` file in the repository root:

    ```hcl
    github_api_token = "YOUR_GITHUB_API_TOKEN"
    route53_zone_id  = "Z008B5091482A026MN9AUQ"
    domain_name      = "sub.example.com"
    ```
  
**Important**: Never commit sensitive data, especially the `github_api_token`, to your repository. Ensure secrets are managed securely.

### Deployment

1. **Setting Up AWS Credentials**:
   Ensure your AWS credentials are properly set up, either by using the `aws configure` command or by setting the necessary environment variables.

2. **Terraform Commands**:
   From the root of the project:

   a. **Planning**:

   ```bash
   terraform plan
   ```

    b. **Deploying Infrastructure and Lambda**:

   ```bash
   terraform apply
   ```

Note: When you run `terraform apply`, Terraform will take care of building the Lambda function from the Go source code and deploying it to AWS.

### DNS Configuration

After successfully applying the Terraform configuration, you will receive an output containing four nameservers. These nameservers are associated with the AWS Route 53 DNS settings for your service.

To complete the setup, you need to configure a subdomain to use these four nameservers. Update your domain provider's DNS settings to point the subdomain to these nameservers.

Ensure that you update the DNS settings in your domain provider's dashboard to use these nameservers for the relevant subdomain.

### API Routes and Curl Usage

This project provides several routes that can be accessed and tested using the `curl` command. Here's a brief guide:

1. **Download Provider Version**:

   ```bash
    curl -X GET https://<your_domain>/v1/providers/{namespace}/{type}/{version}/download/{os}/{arch}
   ```

2. **List Provider Versions**:

   ```bash
    curl -X GET https://<your_domain>/v1/providers/{namespace}/{type}/versions
   ```

3. **List Module Versions**:

   ```bash
    curl -X GET https://<your_domain>/v1/modules/{namespace}/{name}/{system}/versions
   ```

4. **Download Module Version**:

   ```bash
    curl -X GET https://<your_domain>/v1/modules/{namespace}/{name}/{system}/{version}/download
   ```

5. **Terraform Well-Known Metadata**:

   ```bash
    curl -X GET https://<your_domain>/.well-known/terraform.json
   ```

Replace `<your_domain>` with the actual domain where your service is hosted. For dynamic parts of the route, such as `{namespace}` or `{type}`, replace them with appropriate values as per your requirements.

## License

This project is licensed under the terms of the [LICENSE](LICENSE) file.

---

For any additional queries or issues, please open a new issue in the repository.
