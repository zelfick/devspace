---
title: Command - devspace init
sidebar_label: devspace init
id: version-v4.0.1-devspace_init
original_id: devspace_init
---


Initializes DevSpace in the current folder

## Synopsis


```
devspace init [flags]
```

```
#######################################################
#################### devspace init ####################
#######################################################
Initializes a new devspace project within the current
folder. Creates a devspace.yaml with all configuration.
#######################################################
```
## Options

```
      --context string      Context path to use for intialization
      --dockerfile string   Dockerfile to use for initialization (default "./Dockerfile")
  -h, --help                help for init
      --provider string     The cloud provider to use
  -r, --reconfigure         Change existing configuration
```

### Options inherited from parent commands

```
      --debug                 Prints the stack trace if an error occurs
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

## See Also

