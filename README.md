# Registry

A simple golang lambda and supporting terraform to act as a simple, stateless registry that sits on top of github repositories.

## Registry

- [Project Name](#project-name)
  - [Table of Contents](#table-of-contents)
  - [Requirements](#requirements)
  - [Setup](#setup)
  - [Terraform Variables Configuration](#terraform-variables-configuration)
  - [Deployment](#deployment)
  - [DNS Configuration](#dns-configuration)
  - [API Routes and Curl Usage](#api-routes-and-curl-usage)
  - [License](#license)

## Requirements

- **Go**: The AWS Lambda function is written in Go. Ensure you have Go installed.
- **Terraform**: This project uses Terraform for infrastructure management. Make sure to have Terraform installed.
- **AWS CLI**: Ensure that the AWS CLI is installed and configured with the necessary permissions.

## Setup

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

## Terraform Variables Configuration

Before deploying the infrastructure, ensure you've set the required Terraform variables:

- **`github_api_token`**: Personal Access Token (PAT) from GitHub, required for interactions with the GitHub API. [Create a GitHub PAT](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) if you don't have one.

- **`route53_zone_name`**: Name of the Route 53 hosted zone, e.g., "example.com."

- **`domain_name`**: The domain name you wish to manage. This should match or be a subdomain of the `route53_zone_name`.

To provide values for these variables:

- Use the `-var` flag during `terraform apply`, e.g., `terraform apply -var="github_api_token=YOUR_TOKEN"`.
- Or, populate a `terraform.tfvars` file in the repository root:

    ```hcl
    github_api_token = "YOUR_GITHUB_API_TOKEN"
    route53_zone_name = "example.com"
    domain_name       = "sub.example.com"
    ```
  
**Important**: Never commit sensitive data, especially the `github_api_token`, to your repository. Ensure secrets are managed securely.

## Deployment

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

## DNS Configuration

After successfully applying the Terraform configuration, you will receive an output containing four nameservers. These nameservers are associated with the AWS Route 53 DNS settings for your service.

To complete the setup, you need to configure a subdomain to use these four nameservers. Update your domain provider's DNS settings to point the subdomain to these nameservers.

Here's an example of what the Terraform output might look like:

```
Nameservers:
- ns-xxx.awsdns-xx.net.
- ns-xxx.awsdns-xx.org.
- ns-xxx.awsdns-xx.co.uk.
- ns-xxx.awsdns-xx.com.
```

Ensure that you update the DNS settings in your domain provider's dashboard to use these nameservers for the relevant subdomain.

Certainly! Here's a section that briefly describes how to consume these routes using `curl`:

## API Routes and Curl Usage

This project provides several routes that can be accessed and tested using the `curl` command. Here's a brief guide:

1. **Download Provider Version**:
   ```
    curl -X GET https://<your_domain>/v1/providers/{namespace}/{type}/{version}/download/{os}/{arch}
   ```

2. **List Provider Versions**:
   ```
    curl -X GET https://<your_domain>/v1/providers/{namespace}/{type}/versions
   ```

3. **List Module Versions**:
   ```
    curl -X GET https://<your_domain>/v1/modules/{namespace}/{name}/{system}/versions
   ```

4. **Download Module Version**:
   ```
    curl -X GET https://<your_domain>/v1/modules/{namespace}/{name}/{system}/{version}/download
   ```

5. **Terraform Well-Known Metadata**:
   ```
    curl -X GET https://<your_domain>/.well-known/terraform.json
   ```

Replace `<your_domain>` with the actual domain where your service is hosted. For dynamic parts of the route, such as `{namespace}` or `{type}`, replace them with appropriate values as per your requirements.

## License

This project is licensed under the terms of the [LICENSE](LICENSE) file.

---

For any additional queries or issues, please open a new issue in the repository.
```

Please let me know if any additional changes are needed!