# awsole

Open AWS management console from CLI, using assume-role.

## Installation

```
go install github.com/iwataka/awsole@latest
```

## How to use

If you've already assumed a role, run just as follows to open AWS console in your browser.

```
awsole
```

You can specify AWS service you want to open like this.

```
awsole ec2
```

If you've not assumed a role, you can specify it.

```
awsole --role ROLE_TO_ASSUME
```
