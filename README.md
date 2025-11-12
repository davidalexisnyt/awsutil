# AWSUTIL

This is a tool for automating common tasks using the AWS CLI.

I got tired of typing in long strings of AWS CLI commands and having to remember all the options and parameters. Yes, shell scripts can be created to do things like find instance IDs, start SSM sessions, log in to different environments, etc. But I, and other members of my team, need to flip back and forth between Windows, Linux, and MacOS. Having to maintain scripts for PowerShell, bash, zsh, and who knows what else, is a pain.

Thus, the `awsutil` tool was born. The aim is to have the tool minimize the amount of typing, "learn" your most common settings, and generally just help get the job done and get out of the way.

## Features

- **AWS SSO Login**: Simplified login with automatic profile management
- **EC2 Instance Discovery**: Quickly find and list EC2 instances by name pattern
- **SSM Terminal Sessions**: One-command terminal access to EC2 instances
- **Bastion Host Management**:
  - Multiple named bastions per AWS profile
  - Cross-profile bastion lookup by name
  - List configured bastions
  - Interactive configuration with automatic RDS and EC2 discovery
  - Update existing bastion configurations
  - Remove bastion configurations
  - Port forwarding through bastion hosts
  - Auto-assignment of local ports
- **Help System**: Built-in help for all commands
- **Auto-Configuration**: Automatically saves your most-used settings while you use it
- **Cross-Platform**: Single codebase works on Windows, Linux, and macOS

## Prerequisites

This tool automates calls to the AWS CLI, so please ensure that the AWS CLI and the SSM plugin are installed and available in your PATH.

## Installation

This tool is written in Go, so can be compiled for Windows, Mac, or Linux. It requires a single executable and its configuration file. The configuration file is automatically created and managed as you use the tool.

### Step 1: Compile the code

First, make sure you have a [working installation of Go](https://go.dev/dl) on your machine (I always have the very latest version installed). The code has zero package dependencies, so it will compile without any package installation shenanigans.

```shell
go build -ldflags="-s -w"
```

The `-ldflags` option is not required but is nice, since is tells the compiler to produce an executable that is more optimized and does not have debug symbols. The result is a (much) smaller executable than with a plain `go build`.

You should now have an executable (`awsutil.exe` if you're on Windows, `awsutil` otherwise).

### Step 2: Copy/move to a convenient location

It's a good practice to have a `bin` folder in your user home folder that's also in your PATH, but you can put it wherever you'd like. Copy the executable to your chosen location. You're all set!

## Commands

The tool provides the following commands:

- `login` - Log in to AWS SSO
- `instances` - List EC2 instances matching a filter
- `terminal` - Start an SSM terminal session to an EC2 instance
- `bastion` - Start a port forwarding session through a bastion host
- `bastions` - Manage bastion hosts (list, add, update, remove)
- `help` - Show help information (use `awsutil help <command>` for detailed help)
- `docs` - Displays the application documentation (contained in README.md) to the terminal. The markdown is converted and rendered to look beautiful in the terminal.

For detailed help on any command, use:

```shell
awsutil help <command>
```

## Usage

### Logging in to AWS

Let's say we have profiles called `dev` and `prod`, we log in using the `dev` profile with the AWS CLI like this:

```shell
aws sso login --profile dev
```

We can do the same thing with `awsutil`:

```shell
awsutil login -p dev
```

But nothing is gained by using `awsutil` like this, so let's simplify things by automatically taking care of some housekeeping under the hood.

- Once a profile is used with a command, it becomes the default for further commands. e.g. If we login using the dev profile (`awsutil login -p dev`), the `dev` profile becomes the default for other commands like `awsutil instances` or even when you need to login again later, `awsutil login` will log in using the `dev` profile since it was the last one we used.
- There is no need to log in before using another command. `awsutil` will see that we're not currently logged in and will perform the login process before the command that was run. e.g. Let's say we run `awsutil instances myapp` without first running `awsutil login`, we'll first see the AWS login page get launched. Once authentication is done, the `instances` command will be run, listing any existing instances with names starting with "myapp" (we'll go deeper into the `instances` command later).
- Heck, we don't even need to ever run the `awsutil login` command if we don't want to, since ... see the previous bullet point.

### Get a list if EC2 instances

To get a filtered list of EC2 instances, e.g. anything starting with the word "example", we would normally need to run a complex AWS CLI query command like this:

```shell
aws --profile dev ec2 describe-instances \
    --query "Reservations[*].Instances[*].{Instance:InstanceId,AZ:Placement.AvailabilityZone,Name:Tags[?Key=='Name']|[0].Value}" \
    --filters 'Name=tag:Name,Values=example*' \
    --output=table
```

There's no way that anyone will remember this. So let's simplify the process with awsutil:

```shell
awsutil instances --profile dev example
```

But wait... `dev` was already set as our default profile from our previous commands, so we can just use:

```shell
awsutil instances example
```

This will print out something like this:

```
Instances
    example-app-stg-asg: i-0c15ff251abee847f
    ...
```

We're going to work quite a bit with the `i-0c15ff251abee847f` instance, so let's set it as our default. The overriding theme of `awsutil` is automatically making our lives easier. In this case, the `instances` command sees that there is a single EC2 instance matching our query, so it automatically saves the instance info and sets it as our default for commands where we need to use the instance (like the `terminal` command we'll talk about next).

However, if our `instances` query returns more than one EC2 instance, we'll need to specify the instance ID (just once) when we want to connect to it.

> NOTE: You should notice a new file called `awsutil_config.json` in the same location as the `awsutil` executable after running the commands we've gone over so far. Take a look at the file if you're curious to see how `awsutil` keeps track of things.

### Launching an SSM terminal session

We would normally create an SSM session with the AWS CLI like this:

```shell
aws sso login --profile dev
aws ssm --profile dev start-session --target i-0c15ff251abee847f
```

Not as bad as the instance query command, but still too complicated. We have better things to do with our time. With `awsutil`, we would do the same thing like this:

```shell
awsutil terminal
```

Remember the `awsutil instances` command we ran before that returned just one matching instance? The instance information was automatically saved as our default, so the `awsutil terminal` command just knows to connect to it.

There are situations where we might need to specify the instance ID and/or the profile ID with the `terminal` command:

- The last `instances` query we ran returned a list of more than one instance. `awsutil` can't know which one you would want to use, to it does not perform any automatic configuration. In this case, we will need to use `awsutil terminal <instance id>` to connect to the instance we want. NOW, the instance will get automatically saved as our default going forward.
- The last commands we used were against the `dev` instance, but now we want to connect to an instance under the `prod` profile. In this case, we need to specify both the instance ID and the profile: `awsutil terminal -p prod <instance id>`. `awsutil` will automatically save the specified instance ID as the default... for the prod profile. So if we already have a default instance for our dev profile, we still just need to run `awsutil terminal -p dev` to connect to the last dev instance we used.

Again, the theme with `awsutil` is to remember the context of what we were doing so it can save us time and effort.

### Database Bastions

Getting connected to AWS databases through bastion jump hosts can be a messy pain.

- We need to log into the AWS Console web app to gather a bunch of required information that is scattered across a number of different services:
  - Bastion Host Instance ID - the EC2 instance set up to act as our jump host to connect to database resources.
  - Database Host - The instance name of the database host we want to connect to. Except it's not the usual instance name property, but a super long, internal DNS name for the database instance.
  - Databast Host Port - The TCP port for connecting to the database. e.g. Postgres is usually 5432, but can be different.
  - Local Port - This one is easy. It's the port we want our tunnel to use with database drivers in our apps or code to connect to the database. We connect to this port on localhost and the tunnel forwards the traffic to the real database host and port through the EC2 bastion jump host.
- We then need to use the AWS CLI to log in to the environment.
- Finally, we use the AWS CLI's `ssm` command with a lengthy set of arguments.

This is simplified greatly with `awstuil`, as we'll see in a bit.

The `awsutil` bastion functionality supports multiple named bastion tunnels per AWS profile, making it easy to manage connections to different database services.
This is handy for when there are different target databases you would like to access for different applications, even when the environment has a single bastion jump host.

#### Listing Configured Bastions

To view all configured bastions across all profiles:

```shell
awsutil bastions list
```

Or simply:

```shell
awsutil bastions
```

To filter by a specific profile:

```shell
awsutil bastions list -p <profile>
```

This will display all configured bastions, showing:

- Name (with default marker if applicable)
- ID (unique identifier for the bastion)
- Profile (AWS profile the bastion belongs to)
- Instance ID
- Host
- Port
- Local Port

If we have not configured any bastions yet, the `bastions list` results will be a bit boring (i.e. empty). So let's fix that.

#### Bastion Configuration

**Adding a New Bastion**

New bastions are added using the interactive `bastions add` command.

```shell
awsutil bastions add -p <profile>
```

This command will do all the heavy lifting of running multiple AWS commands to gather the information we need to configure our tunnel.

1. If we're not already logged in to the specified profile, `awsutil` will first go through the AWS authentication steps.
2. It runs AWS commands to query for available database instances, then displays the list to us, asking us to select the one we're interested in. The endpoint name and port for the server are saved automatically.
3. It then runs AWS commands to get a list of available EC2 bastion jump hosts, and presents us with the list so we can pick the appropriate one.
4. It asks us for a name for our new bastion tunnel configuration.
5. Finally, we are prompted for the local port we'd like to use. It tries to find the first open port on our machine from port 7000 or above and offers that as the default, which we can override with our own choice.

Done. A new, named bastion configuration is saved under the speficied profile for us to use going forwared.

Now, if we run the `awsutils bastions list` command, we'll see our new bastion in the list.

**Updating an Existing Bastion**

To update an existing bastion configuration, use the `bastions update` command:

```shell
awsutil bastions update -p <profile> --name <bastion-name>
```

Or simply:

```shell
awsutil bastions update -p <profile>
```

This will prompt you for the bastion name if not provided, then guide you through the same interactive process as adding a new bastion. The bastion's ID and profile association are preserved during updates.

**Removing a Bastion**

To remove a bastion configuration, use the `bastions remove` command:

```shell
awsutil bastions remove -p <profile> -n <bastion-name>
```

Or simply:

```shell
awsutil bastions remove -p <profile>
```

This will prompt you for the bastion name if not provided. The command will display the bastion information and ask for confirmation before removing it. If the bastion being removed is the default bastion for the profile, the default will be cleared.

#### Starting a Bastion Session

Once configured, starting a bastion session is simple:

```shell
awsutil bastion
```

This will use the default bastion on the default AWS profile. You can also specify the profile using the `-p` option:

```shell
awsutil bastion -p <profile>
```

This will start the default bastion under the specified profile.

Or we can get really specific and provide both the profile and the name:

```shell
awsutil bastion -p <profile> --name mybastion
```

**Finding Bastions by Name Across Profiles**

If you have multiple bastions configured across different profiles and we use names that are unique across, you can specify the one to use with the `--name` option:

```shell
awsutil bastion --name my-prod-db
```

When using `--name` without specifying a profile:

- The tool first searches for the bastion in the default profile
- If it does not find one, it searches all other profiles
- Once it finds a bastion with the supplied name, it uses the bastion's associated profile to launch the session

If you specify both `--name` and `-p` (or `--profile`), the tool will only search for the bastion in the specified profile:

```shell
awsutil bastion -p dev --name my-db
```

These options give us the flexibility to use `awsutil` in a way that matches our personal approach.

### What if our authentication session has expired?

If we try to issue an AWS CLI command without first logging in, or after our session has expired, we would get a rude response. We would then need to log in and re-attempt our previous command. This is simplified with `awsutil`. If is detects that we don't have a valid authentication session, it will log in with our default profile before executing the command.

So using:

```
awsutil terminal
```

is the same as doing:

```shell
awsutil login
awsutil terminal
```

### More Automatic Configuration Examples

What if you want to start an SSM session to an instance you already know and this is the first time you're using `awsutil`? We can kill two proverbial birds with one stone:

```shell
awsutil terminal --profile dev i-0c15ff251abee847f
```

The profile and instance ID will automatically get saved, an SSO authentication session will be created, and you will be connected to the instance via an SSM terminal session. Then you'd simply use `awsutil terminal` to log in to the instance again later.

Let's simplify even further. If the filter we supply for the `awsutil instances` command results in just one instance being returned, `awsutil` will save that instance ID as your default. Let's try it. First, delete the `awsutil_config.json` file so we're sure we have no defaults saved. Then issue do something like this, where the _filter_ parameter ensures that we get just one instance back:

```shell
awsutil instances --profile spg very-specific-prefix
awsutil terminal
```

Both the profile and the resultant instance ID from the 1st command will be remembered for further commands.

## Getting Help

The tool includes a comprehensive help system. To see all available commands:

```shell
awsutil help
```

For detailed help on a specific command:

```shell
awsutil help bastion
awsutil help bastions
awsutil help bastions list
awsutil help bastions add
awsutil help bastions update
awsutil help bastions remove
awsutil help terminal
# etc.
```

In addition to these help topics, `awsutil` also displays the full documentation (this file):

```shell
awsutil docs
```

## Configuration File Format

The configuration file (`awsutil_config.json`) is stored in the same directory as the executable. It supports:

- **Default Profile**: The AWS CLI profile to use by default
- **Per-Profile Settings**:
  - Default EC2 instance ID
  - Multiple named bastions
  - Default bastion name

Example configuration:

```json
{
  "defaultProfile": "dev",
  "profiles": {
    "dev": {
      "name": "dev",
      "instance": "i-0c15ff251abee847f",
      "bastions": {
        "production-db": {
          "id": "a1b2c3d4e5f6g7h8",
          "name": "production-db",
          "profile": "dev",
          "instance": "i-1234567890abcdef0",
          "host": "prod-db.example.com",
          "port": 5432,
          "localPort": 7000
        }
      },
      "defaultBastion": "production-db"
    },
    "prod": {
      "name": "prod",
      "instance": "i-0987654321fedcba0",
      "bastions": {
        "prod-db": {
          "id": "h8g7f6e5d4c3b2a1",
          "name": "prod-db",
          "profile": "prod",
          "instance": "i-abcdef1234567890",
          "host": "prod-db.example.com",
          "port": 5432,
          "localPort": 7001
        }
      },
      "defaultBastion": "prod-db"
    }
  },
  "bastionLookup": {
    "a1b2c3d4e5f6g7h8": {
      "profile": "dev",
      "name": "production-db"
    },
    "h8g7f6e5d4c3b2a1": {
      "profile": "prod",
      "name": "prod-db"
    }
  }
}
```
