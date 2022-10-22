# vegawallet-dummy

Alternative implementation of the vegawallet CLI for specific use in development and testing.

By default, every incoming transactions will be approved.

## Warning

This software is insecure by design to ease development and testing.

**USE FOR DEVELOPMENT AND TESTING ONLY.**

## Installation

_This requires go 1.19._

At the repository root, run the following command:

```shell
go install
```

## Usage

First, you will to set up you environment using the regular vegawallet CLI.

Once done, instead of using the regular:

```shell
vega wallet service run <ARGS>
```

Use the following command:

```shell
vegawallet-dummy service run <ARGS>
```

To know more about the supported arguments, use the following command:

```shell
vegawallet-dummy service run -h
```
