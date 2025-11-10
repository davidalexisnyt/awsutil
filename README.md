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
  - Port forwarding through bastion hosts
  - Auto-assignment of local ports
- **Help System**: Built-in help for all commands
- **Auto-Configuration**: Automatically saves your most-used settings
- **Cross-Platform**: Single executable works on Windows, Linux, and macOS

## Prerequisites

This tool automates calls to the AWS CLI, so please ensure that the AWS CLI and the SSM plugin are installed and available in your PATH.

## Installation

This tool is written in Go, so can be compiled for Windows, Mac, or Linux. It requires a single executable and its configuration file.

### Step 1: Compile the code

First, make sure you have a [working installation of Go](https://go.dev/dl) on your machine (I always have the very latest version installed). The code has zero package dependencies, so it will compile without any package installation shenanigans.

```shell
go build -ldflags="-s -w"
```

The `-ldflags` option is not required but is nice, since is tells the compiler to produce an executable that is more optimized and does not have debug symbols. The result is a (much) smaller executable than with a plain `go build`.

You should now have an executable (awsutil.exe if you're on Windows, awsutil otherwise).

### Step 2: Copy to a convenient location

It's a good practice to have a `bin` folder in your user home folder that's also in your PATH, but you can put it wherever you'd like. Copy the executable to your chosen location. You're all set!

## Commands

The tool provides the following commands:

- `login` - Log in to AWS SSO
- `instances` - List EC2 instances matching a filter
- `terminal` - Start an SSM terminal session to an EC2 instance
- `bastion` - Start a port forwarding session through a bastion host
- `bastions` - Manage bastion hosts (list, add, update)
- `configure` - Configure default settings
- `help` - Show help information (use `awsutil help <command>` for detailed help)

For detailed help on any command, use:

```shell
awsutil help <command>
```

## Usage/Examples

### Logging in to AWS

Let's say we have profiles called `dev` and `prod`, we log in using the `dev` profile with the AWS CLI like this:

```shell
aws sso login --profile dev
```

or

```shell
aws sso login -p dev
```

We can do the same thing with `awsutil`:

```shell
awsutil login -p dev
```

But nothing is gained by using `awsutil` like this, so let's simplify it by setting `dev` as the default profile for all our commands:

```shell
awsutil configure -p dev
```

Now, we can log in like so:

```shell
awsutil login
```

There are a couple shortcuts for using profiles and logging in that make things more streamlined:

- Once a profile is used with a command, it becomes the default for further commands. e.g. If we login using the dev profile (`awsutil login -p dev`), the dev profile becomes the default for other commands like `awsutil instances ...` or even when you need to login again later (`awsutil login` will log in using the last used profile).
- There is no need to log in before using another command. awsutil will see that we're not currently logged in and will perform the login process before the command that was run. e.g. Let's say we run `awsutil instances bastion` without first running `awsutil login`, we'll first see the AWS login page get launched. Once authentication is done, the `instances` command will be run.

### Get a list if EC2 instances

To get a filtered list of EC2 instances, e.g. anything starting with the word "example", we would need to run a command like this:

```shell
aws --profile dev ec2 describe-instances \
    --query "Reservations[*].Instances[*].{Instance:InstanceId,AZ:Placement.AvailabilityZone,Name:Tags[?Key=='Name']|[0].Value}" \
    --filters 'Name=tag:Name,Values=example*' \
    --output=table
```

We can simplify this:

```shell
awsutil instances --profile dev example
```

But wait... We've already set `dev` as our default profile using `awsutil configure --profile dev` or by previously running a command using the `-p dev` option. So we can just use:

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

Because we've saved our defaults, there will be a new file called `awsutil_config.json` in the same location as the executable. It should look like this:

```json
{
  "defaultProfile": "dev",
  "profiles": {
    "dev": {
      "name": "dev",
      "instance": "i-0c15ff251abee847f",
      "bastions": {},
      "defaultBastion": ""
    }
  }
}
```

Note: The configuration file supports multiple named bastions per profile. Old single-bastion configurations are automatically migrated to the new format.

### Launching an SSM terminal session

We would normally create an SSM session like this:

```shell
aws sso login --profile dev
aws ssm --profile dev start-session --target i-0c15ff251abee847f
```

With `awsutil`, we would do the same thing like this:

```shell
awsutil terminal
```

We can launch a terminal using a different profile by specifying the profile and instance ID if they're not the ones we saved as our defaults:

```shell
awsutil terminal --profile prod i-0c15ff251abee847f
```

Or simply:

```shell
awsutil terminal --profile prod
```

If you've already configured a default instance for the prod profile.

### Database Bastions

The bastion functionality supports multiple named bastion tunnels per AWS profile, making it easy to manage connections to different databases or services.
This is handy for when there are different target databases you would like to access, even when the environment has a single bastion jump host.

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

> Note: By default, `bastions list` shows all bastions across all profiles. Use the `-p` option to filter by a specific profile.

#### Bastion Configuration

**Adding a New Bastion**

New bastions are added using the interactive `bastions add` command:

```shell
awsutil bastions add -p <profile>
```

This command will:

1. Query AWS for available RDS databases
2. Query AWS for available bastion EC2 instances
3. Allow you to interactively select a database and bastion instance
4. Auto-generate a bastion name (or prompt for one)
5. Auto-generate a unique ID for the bastion
6. Auto-find an available local port
7. Save the configuration automatically

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

#### Starting a Bastion Session

Once configured, starting a bastion session is simple:

```shell
awsutil bastion
```

This will use the default bastion on the default AWS profile. You can also specify the profile using the `-p` option:

```shell
awsutil bastion -p <profile>
```

**Finding Bastions by Name Across Profiles**

If you have multiple bastions configured across different profiles, you can specify which one to use with the `--name` option:

```shell
awsutil bastion --name my-db
```

When using `--name` without specifying a profile:

- The tool first searches for the bastion in the default profile
- If not found, it searches all other profiles
- Once found, it uses the bastion's associated profile to launch the session

If you specify both `--name` and `-p` (or `--profile`), the tool will only search for the bastion in the specified profile:

```shell
awsutil bastion -p dev --name my-db
```

This ensures that the correct AWS profile is used for authentication, even when the bastion name exists in multiple profiles.

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
awsutil help terminal
# etc.
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

**Bastion Fields:**

- `id`: Unique identifier for the bastion (auto-generated)
- `name`: Human-readable name for the bastion
- `profile`: AWS profile this bastion is associated with (auto-set)
- `instance`: EC2 instance ID of the bastion host
- `host`: Target hostname or IP address
- `port`: Target port number
- `localPort`: Local port for port forwarding

**BastionLookup:**
The `bastionLookup` map provides fast lookup of bastions by their unique ID, mapping to their profile and name. This is automatically maintained by awsutil.

**Note**: The old single-bastion format (`"bastion": {...}`) is still supported and will be automatically migrated to the new format (`"bastions": {...}`) when the configuration is loaded.
