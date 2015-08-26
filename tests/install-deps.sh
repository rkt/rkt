#!/bin/bash -eu

# Helper script for Continuous Integration Services

if [ "${CI-}" == true ] ; then
	# https://semaphoreci.com/
	if [ "${SEMAPHORE-}" == true ] ; then
		# In Semaphore's last platform update, the default gcc version is 4.6
		# which breaks the systemd build.
		# Set it to 4.8 manually.
		# TODO: remove this when the issue is fixed in Semaphore
		sudo update-alternatives --set gcc /usr/bin/gcc-4.8

		# Most dependencies are already installed on Semaphore.
		# Here we can install any missing dependencies. Whenever
		# Semaphore installs more dependencies on their platform,
		# they should be removed from here to save time.

		# If there is some dependency to install then
		# uncomment the following line and add "sudo apt-get
		# install -y <dep>" after it.

		# sudo apt-get update -qq || true
	fi
fi
