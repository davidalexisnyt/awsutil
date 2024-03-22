# AWSUTIL

This is a tool for automating common tasks using the AWS CLI.
I got tired of typing in long strings of AWS CLI commands and having to remember all the options and parameters. Yes, shell scripts can be created to do things like find instance IDs, start SSM sessions, log in to different environments, etc. But I, and other members of my team, need to flip back and forth between Windows, Linux, and MacOS. Having to maintain scripts for PowerShell, bash, zsh, and who knows what else, is a pain.

Thus, the `awsutil` tool was born. The aim is to have the tool minimize the amount of typing, "learn" your most common settings, and generally just help get the job done and get out of the way.



## Prerequisites

This tool automates calls to the AWS CLI, so please ensure that the AWS CLI and the SSM plugin are installed and available in your PATH.




## Installation

This tool is written in Go, so can be compiled for Windows, Mac, or Linux. It requires a single executable and its configuration file.

### Step 1: Compile the code

First, make sure you have a [working installation of Go](https://go.dev/dl) on your machine (I always have the very latest version installed). The code has zero dependencies, so it will compile without any package installation shenanigans.

```shell
go build
```

You should now have an executable (awsutil.exe if you're on Windows, awsutil otherwise).

### Step 2: Copy to a convenient location

It's a good practice to have a `bin` folder in your user home folder that's also in your PATH, but you can put it wherever you'd like. Copy the executable to your chosen location. You're all set!



## Usage/Examples

### Logging in to AWS

Let's say we have profiles called `dev` and `prod`, we log in using the `dev` profile with the AWS CLI like this:

```shell
aws sso login --profile dev
```

We can do the same thing with `awsutil`:

```shell
awsutil login --profile dev
```

But nothing is gained by using `awsutil` like this, so let's simplify it by setting `dev` as the default profile for all our commands:

```shell
awsutil configure --profile dev
```

Now, we can log in like so:

```shell
awsutil login
```

This gets more useful with other commands.

### Get a list if EC2 instances

To get a filtered list of EC2 instances, e.g. anything starting with the word "example", we would need to run a command like this:

```shell
aws --profile dev ec2 describe-instances --query "Reservations[*].Instances[*].{Instance:InstanceId,AZ:Placement.AvailabilityZone,Name:Tags[?Key=='Name']|[0].Value}" --filters 'Name=tag:Name,Values=example*' --output=table
```

We can simplify this:

```shell
awsutil instances example --profile dev
```

But wait... We've already set `dev` as our default profile using `awsutil configure --profile dev`. So we can just use:

```shell
awsutil instances example
```

This will print out something like this:

```
Instances
    example-app-stg-asg: i-0c15ff251abee847f
    ...
```

We're going to work quite a bit with the `i-0c15ff251abee847f` instance, so let's set it as our default:

```shell
awsutil configure --instance i-0c15ff251abee847f
```



Because we've saved our defaults, there will be a new file called `awsutil_config.json` in the same location as the executable. It should look something like this:

```json
{
    "defaultProfile": "dev",
    "defaultInstance": "i-0c15ff251abee847f"
}
```



### Launching an SSM terminal session

We would normally create an SSM session like this:

```shell
aws sso login --profiledev
aws ssm --profile dev start-session --target i-0c15ff251abee847f
```

With `awsutil`, we would do the same thing like this:

```shell
awsutil terminal
```

We can launch a terminal using a different profile by specifying the profile and instance ID if they're not the ones we saved as our defaults:
```shell
awsutil terminal --profile prod --instance i-0c15ff251abee847f
```

The `awsutil terminal` command needs to know what shell we want, since the Go code needs to ensure that stdin, stdout, and stderr are all redirected properly so the operation is seamless. Make sure you first set the shell with `awsutil configure --shell`. However, if we have not saved any defaults, we can issue the full command like this:

### What if our authentication session has expired?

If we try to issue an AWS CLI command without first logging in, or after our session has expired, we would get a rude response. We'd then need to log in, then re-attempt our command. This is simplified with `awsutil` because it determines if we don't have a valid authentication session and will log in with our default profile before executing the command.

So using:

```
awsutil terminal
```

is the same as doing:

```shell
awsutil login
awsutil terminal
```

### Automatic Configuration

We've used `awsutil configure` to save our default profile and instance ID. What if we forget to do this or don't want to do this up front? If we do something like:

```shell
awsutil instances --profile dev
```

the profile will automatically get saved in the config file, so the next time you want to retrieve instances, you just need to run:

```shell
awsutil instances
```

What if you want to start an SSM session to an instance you already know and this is the first time you're using `awsutil`? We can kill two proverbial birds with one stone:

```shell
awsutil terminal --profile dev i-0c15ff251abee847f
```

The profile and instance ID will automatically get saved, an SSO authentication session will be created, and you will be connected to the instance via an SSM terminal session. Then you'd simply use `awsutil terminal` to log into the instance again later.

Let's simplify even further. If the filter we supply for the `awsutil instances` command results in just one instance being returned, `awsutil` will save that instance ID as your default. Let's try it. First, delete the `awsutil_config.json` file so we're sure we have no defaults saved. Then issue do something like this, where the *filter* parameter ensures that we get just one instance back:

```shell
awsutil instances --profile spg very-specific-prefix
awsutil terminal
```

Both the profile and the resultant instance ID from the 1st command will be remembered for further commands.



## Further Development

- Start a bastion host SSM session
- Save bastion instance ID separately from the "instanceId" for correcting to EC2 instances.

