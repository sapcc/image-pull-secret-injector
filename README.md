Pull secret injector
====================

This repository contains a very simple mutating webhook server that injects image pull secrets into new pods.
For now it injects a static image pull secret into all pods that reference a docker hub image and have no pull secret specified.
The injector takes care of syncing the pull secret to the namespace of the modified pod.

Why ?
=====
Now that Docker has introduced rather strict rate limiting for anonymous pulls this ensures all docker hub images are pulled using an authenticated (payed) account.


Requirements
============
For the example deployment (`make deploy`) the following requirements need to be considered

  * A rather new kustomize binary locally in the PATH
  * cert-manager needs to be installed in the cluster (for the automatic webhook tls setup)

