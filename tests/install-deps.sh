#!/bin/bash -eu

# Helper script for Continuous Integration Services

if [ "${CI-}" == true ] ; then
	# https://semaphoreci.com/
	if [ "${SEMAPHORE-}" == true ] ; then
		# Most dependencies are already installed on Semaphore.
		# Here we can install any missing dependencies. Whenever
		# Semaphore installs more dependencies on their platform,
		# they should be removed from here to save time.

		sudo apt-get update -qq || true
		sudo apt-get install -y libcapture-tiny-perl # used by tools/
	fi
fi
