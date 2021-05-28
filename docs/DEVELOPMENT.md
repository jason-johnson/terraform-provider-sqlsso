# Development Environment Setup

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.12.26+ (to run acceptance tests)
- [Go](https://golang.org/doc/install) 1.16 (to build the provider plugin)

## Quick Start (Visual Studio Code - Dev Containers)

If you wish to work on the provider, you'll first need an environment with [Go](http://www.golang.org) installed on the machine.  If you are working with Visual Studio Code you can use Development Containers to do development in an image which fulfils these requirements.

*Note:* This project uses [Go Modules](https://blog.golang.org/using-go-modules).

Clone repository to your preferred location.

*example Dockerfile*
```
FROM golang:1.16-buster

RUN apt-get update && \
    apt-get install -y exiftool

RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

COPY main.go ./

RUN curl -fsSL https://apt.releases.hashicorp.com/gpg | apt-key add -
RUN apt-get install -y software-properties-common
RUN apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
RUN apt-get update && apt-get install -y terraform

ENTRYPOINT ["go", "run" "main.go"]
```

Follow the instructions [here](https://code.visualstudio.com/docs/remote/containers-tutorial) but select "Existing Dockerfile" instead of the container suggested on the page.  Once inside the container you can create a bash shell and run `make`.  This will build the provider and run the tests.

## Testing the Provider

In order to test the provider, you can run `make`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run. Please read [Running and Writing Acceptance Tests](contributing/running-and-writing-acceptance-tests.md) in the contribution guidelines for more information on usage.

*Note:* Currently this does not work as a working Azure Environment would be required to test.  See issue [#3](https://github.com/jason-johnson/terraform-provider-sqlsso/issues/3) for details.

```sh
$ make testacc
```

## Using the Provider

With Terraform v0.14 and later, [development overrides for provider developers](https://www.terraform.io/docs/cli/config/config-file.html#development-overrides-for-provider-developers) can be leveraged in order to use the provider built from source.

To do this, populate a Terraform CLI configuration file (`~/.terraformrc` for all platforms other than Windows; `terraform.rc` in the `%APPDATA%` directory when using Windows) with at least the following options:

```hcl
provider_installation {
  dev_overrides {
    "jason-johnson/sqlsso" = "[REPLACE WITH GOPATH]/bin"
  }
  direct {}
}
```

## Debugging

When running inside the Dev container you can do full debugging by setting a break point and debugging Go in the normal way.  This will output a string and explain which environment variable to set it in.  Once this is done you can run any terraform command and you breakpoints will hit as normal.